package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Category is a user-defined expense category. ColorSlot (1..8) picks one of
// the design system's category colours.
type Category struct {
	ID        int64
	Name      string
	ColorSlot int
	CreatedAt time.Time
}

// Expense is a single recorded expense. AmountMinor is stored in minor units
// (cents/kopecks) — never a float.
type Expense struct {
	ID           int64
	AmountMinor  int64
	CategoryID   int64
	CategoryName string
	ColorSlot    int
	Date         time.Time
	Comment      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CategoryTotal is one row of the "by category" breakdown.
type CategoryTotal struct {
	CategoryID int64
	Name       string
	ColorSlot  int
	TotalMinor int64
	Count      int
	Percent    float64 // share of the period total, shown as a number
	BarPercent float64 // share of the largest category: the width of the bar
	DashLength float64 // pre-computed donut segment length
	DashOffset float64
	IsOther    bool
}

// ErrDuplicateCategory is returned when a category name is already taken.
var ErrDuplicateCategory = errors.New("category already exists")

// ErrCategoryInUse is returned when deleting a category that has expenses.
var ErrCategoryInUse = errors.New("category is in use")

// Store owns the database connection and holds every SQL query in the app.
type Store struct{ db *sql.DB }

const schema = `
CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    name_lower TEXT    NOT NULL,
    color_slot INTEGER NOT NULL DEFAULT 1,
    created_at TEXT    NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_name ON categories(name_lower);

CREATE TABLE IF NOT EXISTS expenses (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    category_id  INTEGER NOT NULL REFERENCES categories(id),
    date         TEXT    NOT NULL,
    comment      TEXT    NOT NULL DEFAULT '',
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses(category_id);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`

// OpenStore opens (and creates on first run) the SQLite database at path,
// applies the schema and returns a ready store.
func OpenStore(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create data directory: %w", err)
		}
	}

	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// One writer at a time keeps SQLite happy and is plenty for one user.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close releases the database connection.
func (s *Store) Close() error { return s.db.Close() }

// ---------- Settings ----------

// Settings holds the whole app configuration a user can change.
type Settings struct {
	Currency string
	Lang     string
	Theme    string
}

// DefaultSettings is what a fresh database starts with.
var DefaultSettings = Settings{Currency: "EUR", Lang: LangRU, Theme: "system"}

// Settings reads the stored settings, falling back to the defaults.
func (s *Store) Settings() (Settings, error) {
	out := DefaultSettings
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return out, err
		}
		switch k {
		case "currency":
			if IsCurrency(v) {
				out.Currency = v
			}
		case "lang":
			if IsLang(v) {
				out.Lang = v
			}
		case "theme":
			if v == "light" || v == "dark" || v == "system" {
				out.Theme = v
			}
		}
	}
	return out, rows.Err()
}

// SetSetting stores one setting value.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// ---------- Categories ----------

// Categories returns every category, ordered by name.
func (s *Store) Categories() ([]Category, error) {
	rows, err := s.db.Query(
		`SELECT id, name, color_slot, created_at FROM categories ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Category
	for rows.Next() {
		var c Category
		var created string
		if err := rows.Scan(&c.ID, &c.Name, &c.ColorSlot, &created); err != nil {
			return nil, err
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, c)
	}
	return out, rows.Err()
}

// Category returns one category by id.
func (s *Store) Category(id int64) (Category, error) {
	var c Category
	var created string
	err := s.db.QueryRow(
		`SELECT id, name, color_slot, created_at FROM categories WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.ColorSlot, &created)
	if err != nil {
		return c, err
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return c, nil
}

// CreateCategory adds a category and assigns it the next colour slot.
// It returns ErrDuplicateCategory when the name is already taken
// (case-insensitively, including Cyrillic).
func (s *Store) CreateCategory(name string) (int64, error) {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)

	var exists int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM categories WHERE name_lower = ?`, lower).
		Scan(&exists); err != nil {
		return 0, err
	}
	if exists > 0 {
		return 0, ErrDuplicateCategory
	}

	// Colour slots cycle through 1..8 so the donut stays readable.
	var count int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&count); err != nil {
		return 0, err
	}
	slot := int(count%8) + 1

	res, err := s.db.Exec(
		`INSERT INTO categories (name, name_lower, color_slot, created_at) VALUES (?, ?, ?, ?)`,
		name, lower, slot, time.Now().Format(time.RFC3339))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return 0, ErrDuplicateCategory
		}
		return 0, err
	}
	return res.LastInsertId()
}

// CategoryExpenseCount counts the expenses that use a category.
func (s *Store) CategoryExpenseCount(id int64) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM expenses WHERE category_id = ?`, id).Scan(&n)
	return n, err
}

// DeleteCategory removes a category. A category still used by expenses is kept
// and ErrCategoryInUse is returned — expenses never lose their category.
func (s *Store) DeleteCategory(id int64) error {
	n, err := s.CategoryExpenseCount(id)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrCategoryInUse
	}
	_, err = s.db.Exec(`DELETE FROM categories WHERE id = ?`, id)
	return err
}

// ---------- Expenses ----------

const expenseColumns = `e.id, e.amount_minor, e.category_id, c.name, c.color_slot,
                        e.date, e.comment, e.created_at, e.updated_at`

func scanExpense(rows interface{ Scan(...any) error }) (Expense, error) {
	var e Expense
	var date, created, updated string
	err := rows.Scan(&e.ID, &e.AmountMinor, &e.CategoryID, &e.CategoryName, &e.ColorSlot,
		&date, &e.Comment, &created, &updated)
	if err != nil {
		return e, err
	}
	e.Date, _ = ParseDate(date)
	e.CreatedAt, _ = time.Parse(time.RFC3339, created)
	e.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return e, nil
}

// CreateExpense stores a new expense and returns its id.
func (s *Store) CreateExpense(amountMinor int64, categoryID int64, date time.Time, comment string) (int64, error) {
	now := time.Now().Format(time.RFC3339)
	res, err := s.db.Exec(
		`INSERT INTO expenses (amount_minor, category_id, date, comment, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		amountMinor, categoryID, FormatISO(date), comment, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateExpense saves changes to an existing expense.
func (s *Store) UpdateExpense(id int64, amountMinor int64, categoryID int64, date time.Time, comment string) error {
	res, err := s.db.Exec(
		`UPDATE expenses SET amount_minor = ?, category_id = ?, date = ?, comment = ?, updated_at = ?
		 WHERE id = ?`,
		amountMinor, categoryID, FormatISO(date), comment, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteExpense removes one expense.
func (s *Store) DeleteExpense(id int64) error {
	res, err := s.db.Exec(`DELETE FROM expenses WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Expense returns one expense with its category.
func (s *Store) Expense(id int64) (Expense, error) {
	row := s.db.QueryRow(
		`SELECT `+expenseColumns+` FROM expenses e JOIN categories c ON c.id = e.category_id
		 WHERE e.id = ?`, id)
	return scanExpense(row)
}

// RecentExpenses returns the newest expenses, most recent date first.
func (s *Store) RecentExpenses(limit int) ([]Expense, error) {
	rows, err := s.db.Query(
		`SELECT `+expenseColumns+` FROM expenses e JOIN categories c ON c.id = e.category_id
		 ORDER BY e.date DESC, e.created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectExpenses(rows)
}

// ExpensesBetween returns the expenses of a date range, newest first.
// Both ends are inclusive.
func (s *Store) ExpensesBetween(from, to time.Time) ([]Expense, error) {
	rows, err := s.db.Query(
		`SELECT `+expenseColumns+` FROM expenses e JOIN categories c ON c.id = e.category_id
		 WHERE e.date BETWEEN ? AND ?
		 ORDER BY e.date DESC, e.created_at DESC`, FormatISO(from), FormatISO(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectExpenses(rows)
}

// ExpensesBetweenAsc returns the expenses of a range oldest first — the order
// used by the CSV export.
func (s *Store) ExpensesBetweenAsc(from, to time.Time) ([]Expense, error) {
	rows, err := s.db.Query(
		`SELECT `+expenseColumns+` FROM expenses e JOIN categories c ON c.id = e.category_id
		 WHERE e.date BETWEEN ? AND ?
		 ORDER BY e.date ASC, e.created_at ASC`, FormatISO(from), FormatISO(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectExpenses(rows)
}

// CategoryExpensesBetween returns one category's expenses in a range.
func (s *Store) CategoryExpensesBetween(categoryID int64, from, to time.Time) ([]Expense, error) {
	rows, err := s.db.Query(
		`SELECT `+expenseColumns+` FROM expenses e JOIN categories c ON c.id = e.category_id
		 WHERE e.category_id = ? AND e.date BETWEEN ? AND ?
		 ORDER BY e.date DESC, e.created_at DESC`,
		categoryID, FormatISO(from), FormatISO(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectExpenses(rows)
}

func collectExpenses(rows *sql.Rows) ([]Expense, error) {
	var out []Expense
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SumBetween returns the total and the number of expenses in a date range.
func (s *Store) SumBetween(from, to time.Time) (total int64, count int, err error) {
	err = s.db.QueryRow(
		`SELECT COALESCE(SUM(amount_minor), 0), COUNT(*) FROM expenses WHERE date BETWEEN ? AND ?`,
		FormatISO(from), FormatISO(to)).Scan(&total, &count)
	return
}

// TotalsByCategory groups the expenses of a range by category, largest first.
func (s *Store) TotalsByCategory(from, to time.Time) ([]CategoryTotal, error) {
	rows, err := s.db.Query(
		`SELECT c.id, c.name, c.color_slot, SUM(e.amount_minor), COUNT(*)
		 FROM expenses e JOIN categories c ON c.id = e.category_id
		 WHERE e.date BETWEEN ? AND ?
		 GROUP BY c.id, c.name, c.color_slot
		 ORDER BY SUM(e.amount_minor) DESC, c.name COLLATE NOCASE`,
		FormatISO(from), FormatISO(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CategoryTotal
	for rows.Next() {
		var t CategoryTotal
		if err := rows.Scan(&t.CategoryID, &t.Name, &t.ColorSlot, &t.TotalMinor, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
