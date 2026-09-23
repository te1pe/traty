package main

import (
	"path/filepath"
	"testing"
	"time"
)

// testStore opens a throwaway database in a temporary directory, proving along
// the way that a fresh database is created and migrated automatically.
func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

// mustCategory creates a category and fails the test if it cannot.
func mustCategory(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	id, err := s.CreateCategory(name)
	if err != nil {
		t.Fatalf("CreateCategory(%q): %v", name, err)
	}
	return id
}

func TestCreateAndReadExpense(t *testing.T) {
	s := testStore(t)
	catID := mustCategory(t, s, "Еда")

	date := day(2026, 9, 22)
	id, err := s.CreateExpense(125050, catID, date, "Ужин с друзьями")
	if err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	got, err := s.Expense(id)
	if err != nil {
		t.Fatalf("Expense: %v", err)
	}
	if got.AmountMinor != 125050 {
		t.Errorf("amount = %d, want 125050", got.AmountMinor)
	}
	if got.CategoryName != "Еда" {
		t.Errorf("category = %q, want Еда", got.CategoryName)
	}
	if !got.Date.Equal(date) {
		t.Errorf("date = %s, want %s", FormatISO(got.Date), FormatISO(date))
	}
	if got.Comment != "Ужин с друзьями" {
		t.Errorf("comment = %q", got.Comment)
	}
}

func TestDuplicateCategoryIsRejected(t *testing.T) {
	s := testStore(t)
	mustCategory(t, s, "Еда")

	// Same name in a different case — including Cyrillic — must be rejected.
	if _, err := s.CreateCategory("еда"); err != ErrDuplicateCategory {
		t.Errorf("CreateCategory(еда) = %v, want ErrDuplicateCategory", err)
	}
	if _, err := s.CreateCategory("ЕДА"); err != ErrDuplicateCategory {
		t.Errorf("CreateCategory(ЕДА) = %v, want ErrDuplicateCategory", err)
	}
}

func TestDeleteCategoryInUse(t *testing.T) {
	s := testStore(t)
	catID := mustCategory(t, s, "Еда")
	if _, err := s.CreateExpense(1000, catID, day(2026, 9, 22), ""); err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	if err := s.DeleteCategory(catID); err != ErrCategoryInUse {
		t.Fatalf("DeleteCategory of a used category = %v, want ErrCategoryInUse", err)
	}

	// Once the expense is gone, the category can be removed.
	items, _ := s.RecentExpenses(1)
	if err := s.DeleteExpense(items[0].ID); err != nil {
		t.Fatalf("DeleteExpense: %v", err)
	}
	if err := s.DeleteCategory(catID); err != nil {
		t.Fatalf("DeleteCategory after clearing: %v", err)
	}
}

// Expense validation goes through the same form the handlers use.
func TestExpenseFormValidation(t *testing.T) {
	s := testStore(t)
	catID := mustCategory(t, s, "Еда")
	today := day(2026, 9, 22)

	cases := []struct {
		name      string
		form      ExpenseForm
		wantField string
	}{
		{"empty amount", ExpenseForm{Amount: "", CategoryID: catID, DatePreset: "today"}, "amount"},
		{"zero amount", ExpenseForm{Amount: "0", CategoryID: catID, DatePreset: "today"}, "amount"},
		{"not a number", ExpenseForm{Amount: "abc", CategoryID: catID, DatePreset: "today"}, "amount"},
		{"no category", ExpenseForm{Amount: "10", DatePreset: "today"}, "category"},
		{"unknown category", ExpenseForm{Amount: "10", CategoryID: 999, DatePreset: "today"}, "category"},
		{"future date", ExpenseForm{Amount: "10", CategoryID: catID, DatePreset: "custom", Date: "2027-01-01"}, "date"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := c.form
			f.Errors = map[string]string{}
			if _, _, _, ok := f.validated(s, LangRU, today); ok {
				t.Fatalf("form accepted, want rejection on %q", c.wantField)
			}
			if f.Err(c.wantField) == "" {
				t.Errorf("no error on %q, got %v", c.wantField, f.Errors)
			}
		})
	}

	// A valid form passes and keeps the values.
	f := ExpenseForm{Amount: "1250,50", CategoryID: catID, DatePreset: "today", Errors: map[string]string{}}
	amount, cat, date, ok := f.validated(s, LangRU, today)
	if !ok {
		t.Fatalf("valid form rejected: %v", f.Errors)
	}
	if amount != 125050 || cat != catID || !date.Equal(today) {
		t.Errorf("validated = (%d, %d, %s), want (125050, %d, %s)",
			amount, cat, FormatISO(date), catID, FormatISO(today))
	}
}

// A brand-new category typed in the expense form is created along with it.
func TestExpenseFormCreatesNewCategory(t *testing.T) {
	s := testStore(t)
	today := day(2026, 9, 22)

	f := ExpenseForm{Amount: "12,50", NewCategory: "Кафе", DatePreset: "today", Errors: map[string]string{}}
	_, catID, _, ok := f.validated(s, LangRU, today)
	if !ok {
		t.Fatalf("form rejected: %v", f.Errors)
	}

	cat, err := s.Category(catID)
	if err != nil {
		t.Fatalf("Category: %v", err)
	}
	if cat.Name != "Кафе" {
		t.Errorf("created category = %q, want Кафе", cat.Name)
	}
}

func TestTodayTotal(t *testing.T) {
	s := testStore(t)
	catID := mustCategory(t, s, "Еда")
	today := day(2026, 9, 22)

	s.CreateExpense(4000, catID, today, "")
	s.CreateExpense(550, catID, today, "")
	s.CreateExpense(10000, catID, today.AddDate(0, 0, -1), "") // yesterday, must not count

	total, count, err := s.SumBetween(today, today)
	if err != nil {
		t.Fatalf("SumBetween: %v", err)
	}
	if total != 4550 {
		t.Errorf("today total = %d, want 4550", total)
	}
	if count != 2 {
		t.Errorf("today count = %d, want 2", count)
	}
}

// The week runs Monday → today, not the last seven days.
func TestWeekTotalFromMonday(t *testing.T) {
	s := testStore(t)
	catID := mustCategory(t, s, "Еда")

	wednesday := day(2026, 9, 23)
	monday := day(2026, 9, 21)
	tuesday := day(2026, 9, 22)
	lastFriday := day(2026, 9, 18) // previous week: excluded

	s.CreateExpense(1000, catID, monday, "")
	s.CreateExpense(2000, catID, tuesday, "")
	s.CreateExpense(3000, catID, wednesday, "")
	s.CreateExpense(9999, catID, lastFriday, "")

	from := WeekStart(wednesday)
	if !from.Equal(monday) {
		t.Fatalf("WeekStart(Wed) = %s, want %s", FormatISO(from), FormatISO(monday))
	}

	total, count, err := s.SumBetween(from, wednesday)
	if err != nil {
		t.Fatalf("SumBetween: %v", err)
	}
	if total != 6000 {
		t.Errorf("week total = %d, want 6000 (Mon+Tue+Wed only)", total)
	}
	if count != 3 {
		t.Errorf("week count = %d, want 3", count)
	}
}

func TestReportForCustomRange(t *testing.T) {
	s := testStore(t)
	food := mustCategory(t, s, "Еда")
	transport := mustCategory(t, s, "Транспорт")

	s.CreateExpense(45050, food, day(2026, 9, 1), "")
	s.CreateExpense(18000, transport, day(2026, 9, 15), "")
	s.CreateExpense(31500, food, day(2026, 9, 22), "")
	s.CreateExpense(50000, food, day(2026, 8, 31), "") // before the range
	s.CreateExpense(70000, food, day(2026, 9, 23), "") // after the range

	from, to := day(2026, 9, 1), day(2026, 9, 22)

	total, count, err := s.SumBetween(from, to)
	if err != nil {
		t.Fatalf("SumBetween: %v", err)
	}
	if total != 94550 {
		t.Errorf("report total = %d, want 94550", total)
	}
	if count != 3 {
		t.Errorf("report count = %d, want 3", count)
	}

	items, err := s.ExpensesBetween(from, to)
	if err != nil {
		t.Fatalf("ExpensesBetween: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("report list has %d rows, want 3", len(items))
	}
}

func TestTotalsByCategory(t *testing.T) {
	s := testStore(t)
	food := mustCategory(t, s, "Еда")
	transport := mustCategory(t, s, "Транспорт")
	shop := mustCategory(t, s, "Покупки")

	from, to := day(2026, 9, 1), day(2026, 9, 22)
	s.CreateExpense(30000, food, day(2026, 9, 2), "")
	s.CreateExpense(15050, food, day(2026, 9, 10), "")
	s.CreateExpense(18000, transport, day(2026, 9, 11), "")
	s.CreateExpense(31500, shop, day(2026, 9, 12), "")
	s.CreateExpense(99900, food, day(2026, 9, 30), "") // outside the range

	totals, err := s.TotalsByCategory(from, to)
	if err != nil {
		t.Fatalf("TotalsByCategory: %v", err)
	}
	if len(totals) != 3 {
		t.Fatalf("got %d categories, want 3", len(totals))
	}

	// Largest first.
	want := []struct {
		name  string
		total int64
		count int
	}{
		{"Еда", 45050, 2},
		{"Покупки", 31500, 1},
		{"Транспорт", 18000, 1},
	}
	for i, w := range want {
		if totals[i].Name != w.name || totals[i].TotalMinor != w.total || totals[i].Count != w.count {
			t.Errorf("row %d = (%s, %d, %d), want (%s, %d, %d)",
				i, totals[i].Name, totals[i].TotalMinor, totals[i].Count, w.name, w.total, w.count)
		}
	}
}

// Editing an expense must move its amount into the new day's totals.
func TestUpdateExpenseUpdatesTotals(t *testing.T) {
	s := testStore(t)
	food := mustCategory(t, s, "Еда")
	today := day(2026, 9, 22)

	id, err := s.CreateExpense(1000, food, today, "")
	if err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	if err := s.UpdateExpense(id, 2500, food, today.AddDate(0, 0, -1), "изменено"); err != nil {
		t.Fatalf("UpdateExpense: %v", err)
	}

	total, count, _ := s.SumBetween(today, today)
	if total != 0 || count != 0 {
		t.Errorf("today after moving the expense = (%d, %d), want (0, 0)", total, count)
	}

	total, count, _ = s.SumBetween(today.AddDate(0, 0, -1), today)
	if total != 2500 || count != 1 {
		t.Errorf("two-day total = (%d, %d), want (2500, 1)", total, count)
	}
}

// The donut folds everything past the fifth category into "Other".
func TestBuildDonutFoldsTail(t *testing.T) {
	totals := []CategoryTotal{
		{Name: "a", TotalMinor: 5000}, {Name: "b", TotalMinor: 2000},
		{Name: "c", TotalMinor: 1500}, {Name: "d", TotalMinor: 1000},
		{Name: "e", TotalMinor: 300}, {Name: "f", TotalMinor: 200},
	}
	var total int64
	for _, x := range totals {
		total += x.TotalMinor
	}

	segments := BuildDonut(totals, total, "Остальные")
	if len(segments) != maxDonutSegments {
		t.Fatalf("got %d segments, want %d", len(segments), maxDonutSegments)
	}
	last := segments[len(segments)-1]
	if !last.IsOther {
		t.Errorf("last segment is %q, want the Other segment", last.Name)
	}
	if last.TotalMinor != 500 {
		t.Errorf("Other total = %d, want 500 (300+200)", last.TotalMinor)
	}

	var sum float64
	for _, seg := range segments {
		sum += seg.Percent
	}
	if sum < 99.9 || sum > 100.1 {
		t.Errorf("percentages add up to %.2f, want 100", sum)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	s := testStore(t)

	got, err := s.Settings()
	if err != nil {
		t.Fatalf("Settings: %v", err)
	}
	if got != DefaultSettings {
		t.Errorf("fresh database settings = %+v, want %+v", got, DefaultSettings)
	}

	for key, value := range map[string]string{"currency": "RUB", "lang": "en", "theme": "dark"} {
		if err := s.SetSetting(key, value); err != nil {
			t.Fatalf("SetSetting(%s): %v", key, err)
		}
	}
	got, _ = s.Settings()
	if got.Currency != "RUB" || got.Lang != "en" || got.Theme != "dark" {
		t.Errorf("stored settings = %+v", got)
	}

	// A bad value stored somehow must not break the app.
	s.SetSetting("currency", "XXX")
	got, _ = s.Settings()
	if got.Currency != DefaultSettings.Currency {
		t.Errorf("invalid currency = %q, want the default", got.Currency)
	}
}
