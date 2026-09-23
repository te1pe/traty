package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// maxCommentLen and maxCategoryNameLen match the design system's limits.
const (
	maxCommentLen      = 120
	maxCategoryNameLen = 32
)

// newCategoryValue is the radio value of the "New" chip in the category picker.
const newCategoryValue = "__new__"

// ExpenseForm is both the input of an expense form and what the template draws
// again when validation fails, so nothing the user typed is lost.
type ExpenseForm struct {
	ID          int64
	Amount      string
	CategoryID  int64
	NewCategory string
	DatePreset  string // today | yesterday | custom
	Date        string // YYYY-MM-DD
	Comment     string
	ReturnTo    string
	Errors      map[string]string
}

// Err returns the message for one field, if any.
func (f *ExpenseForm) Err(field string) string {
	if f == nil {
		return ""
	}
	return f.Errors[field]
}

// HasErrors reports whether the form failed validation.
func (f *ExpenseForm) HasErrors() bool { return f != nil && len(f.Errors) > 0 }

// IsNewCategory reports whether the "New" chip is the selected one.
func (f *ExpenseForm) IsNewCategory() bool {
	return f != nil && f.CategoryID == 0 && f.NewCategory != ""
}

// CategoryValue renders the value that should be checked in the picker.
func (f *ExpenseForm) CategoryValue() string {
	if f == nil {
		return ""
	}
	if f.CategoryID > 0 {
		return strconv.FormatInt(f.CategoryID, 10)
	}
	if f.NewCategory != "" {
		return newCategoryValue
	}
	return ""
}

// expenseFormFrom reads an expense form out of a POST request.
func expenseFormFrom(r *http.Request) *ExpenseForm {
	f := &ExpenseForm{
		Amount:      strings.TrimSpace(r.PostFormValue("amount")),
		NewCategory: strings.TrimSpace(r.PostFormValue("new_category")),
		DatePreset:  r.PostFormValue("date_preset"),
		Date:        strings.TrimSpace(r.PostFormValue("date")),
		Comment:     strings.TrimSpace(r.PostFormValue("comment")),
		ReturnTo:    safeReturn(r.PostFormValue("return_to"), "/"),
		Errors:      map[string]string{},
	}
	if v := r.PostFormValue("category"); v != "" && v != newCategoryValue {
		f.CategoryID = parseID(v)
	}
	return f
}

// validated checks the form and, when it is valid, returns the values ready to
// store. Creating a brand-new category is part of saving, so the store is
// needed here; the category is only created once everything else is valid.
func (f *ExpenseForm) validated(s *Store, lang string, today time.Time) (amount int64, categoryID int64, date time.Time, ok bool) {
	// Amount.
	amount, err := ParseAmount(f.Amount)
	switch err {
	case nil:
	case ErrAmountEmpty:
		f.Errors["amount"] = T(lang, "error.amount_empty")
	case ErrAmountZero:
		f.Errors["amount"] = T(lang, "error.amount_zero")
	case ErrAmountTooBig:
		f.Errors["amount"] = T(lang, "error.amount_big")
	default:
		f.Errors["amount"] = T(lang, "error.amount_shape")
	}

	// Date: a preset or a picked day, never in the future.
	switch f.DatePreset {
	case "today", "":
		date = today
		f.DatePreset = "today"
	case "yesterday":
		date = today.AddDate(0, 0, -1)
	default:
		f.DatePreset = "custom"
		d, derr := ParseDate(f.Date)
		if derr != nil {
			f.Errors["date"] = T(lang, "error.date_shape")
		} else if d.After(today) {
			f.Errors["date"] = T(lang, "error.date_future")
			date = d
		} else {
			date = d
		}
	}
	if f.Date == "" && f.Errors["date"] == "" {
		f.Date = FormatISO(date)
	}

	// Category: an existing one, or a new one created along with the expense.
	switch {
	case f.CategoryID > 0:
		if _, cerr := s.Category(f.CategoryID); cerr != nil {
			f.Errors["category"] = T(lang, "error.category_empty")
		} else {
			categoryID = f.CategoryID
		}
	case f.NewCategory != "":
		name := f.NewCategory
		if len([]rune(name)) > maxCategoryNameLen {
			f.Errors["category"] = T(lang, "error.name_long")
		}
	default:
		f.Errors["category"] = T(lang, "error.category_empty")
	}

	if len([]rune(f.Comment)) > maxCommentLen {
		f.Errors["comment"] = T(lang, "error.note_long")
	}

	if len(f.Errors) > 0 {
		return 0, 0, date, false
	}

	// Everything else is valid: create the new category now, if asked for.
	if categoryID == 0 {
		id, cerr := s.CreateCategory(f.NewCategory)
		if cerr == ErrDuplicateCategory {
			// The name exists — reuse that category instead of failing.
			cats, lerr := s.Categories()
			if lerr != nil {
				f.Errors["category"] = T(lang, "error.category_dup")
				return 0, 0, date, false
			}
			lower := strings.ToLower(f.NewCategory)
			for _, c := range cats {
				if strings.ToLower(c.Name) == lower {
					id = c.ID
					break
				}
			}
			if id == 0 {
				f.Errors["category"] = T(lang, "error.category_dup")
				return 0, 0, date, false
			}
		} else if cerr != nil {
			return 0, 0, date, false
		}
		categoryID = id
	}

	return amount, categoryID, date, true
}

// fillFrom prepares the form for editing an existing expense.
func expenseFormOf(e Expense, today time.Time, returnTo string) *ExpenseForm {
	preset := "custom"
	switch {
	case e.Date.Equal(today):
		preset = "today"
	case e.Date.Equal(today.AddDate(0, 0, -1)):
		preset = "yesterday"
	}
	return &ExpenseForm{
		ID:         e.ID,
		Amount:     FormatAmountInput(e.AmountMinor),
		CategoryID: e.CategoryID,
		DatePreset: preset,
		Date:       FormatISO(e.Date),
		Comment:    e.Comment,
		ReturnTo:   returnTo,
		Errors:     map[string]string{},
	}
}

// parseID reads a positive int64 id, returning 0 when the value is not one.
func parseID(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
