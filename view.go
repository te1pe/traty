package main

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

// templates holds every page, parsed once at startup.
var templates *template.Template

// funcs are the few helpers the templates need beyond the Page methods.
var funcs = template.FuncMap{
	"f1":  func(v float64) string { return fmt.Sprintf("%.1f", v) },
	"f2":  func(v float64) string { return fmt.Sprintf("%.2f", v) },
	"pct": func(v float64) string { return fmt.Sprintf("%.0f", v) },
	"iso": FormatISO,
	"add": func(a, b int) int { return a + b },
	// dict builds a map so a partial can take several values at once.
	"dict": func(values ...any) map[string]any {
		m := make(map[string]any, len(values)/2)
		for i := 0; i+1 < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				continue
			}
			m[key] = values[i+1]
		}
		return m
	},
}

// parseTemplates loads the templates from the embedded files.
func parseTemplates() error {
	t, err := template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return err
	}
	templates = t
	return nil
}

// Page carries everything a layout needs, plus the formatting helpers the
// templates call. Every page struct embeds it.
type Page struct {
	Lang           string
	Theme          string
	Currency       string
	Nav            string // which nav item is current: overview, reports, categories, settings
	Path           string // current path, used as return_to in the forms
	ActiveCategory int64  // category whose sheet is open, highlighted on the donut
	Title          string
	Today          time.Time
	Categories     []Category // for the "add expense" sheet present on every page
	Toast          string
	NewID          int64 // row to highlight after a save
	NewForm        *ExpenseForm
}

// T translates a key into the page language.
func (p Page) T(key string) string { return T(p.Lang, key) }

// Tf translates and formats a key.
func (p Page) Tf(key string, args ...any) string { return Tf(p.Lang, key, args...) }

// Money splits an amount into whole/decimal/sign parts for the large styles.
func (p Page) Money(minor int64) MoneyParts { return SplitMoney(minor, p.Currency) }

// Amount renders a one-line amount: "1 245,50 €".
func (p Page) Amount(minor int64) string { return FormatMoney(minor, p.Currency) }

// Symbol is the currency sign shown next to the amount field.
func (p Page) Symbol() string { return CurrencySymbol(p.Currency) }

// DayRelative renders "Сегодня" / "Вчера" / "22 сент.".
func (p Page) DayRelative(d time.Time) string { return FormatDayRelative(d, p.Today, p.Lang) }

// DayShort renders "22 сент." / "22 Sept".
func (p Page) DayShort(d time.Time) string { return FormatDayShort(d, p.Lang) }

// DayLong renders "22 сентября" / "22 September".
func (p Page) DayLong(d time.Time) string { return FormatDayLong(d, p.Lang) }

// WeekdayDay renders "вторник, 22 сентября".
func (p Page) WeekdayDay(d time.Time) string { return FormatWeekdayDay(d, p.Lang) }

// WeekdaySpan renders "пн — вт".
func (p Page) WeekdaySpan(from, to time.Time) string { return FormatWeekdaySpan(from, to, p.Lang) }

// Range renders a report period.
func (p Page) Range(from, to time.Time) string { return FormatRange(from, to, p.Lang) }

// Plural renders "3 расхода" / "3 expenses".
func (p Page) Plural(n int) string { return PluralExpenses(p.Lang, n) }

// ExpenseWord is the bare noun shown under the donut.
func (p Page) ExpenseWord(n int) string { return ExpenseWord(p.Lang, n) }

// ISO renders a date for a form field or a URL.
func (p Page) ISO(d time.Time) string { return FormatISO(d) }

// TodayISO is the max value of every date input.
func (p Page) TodayISO() string { return FormatISO(p.Today) }

// AddForm is the blank (or refilled) form of the "new expense" sheet.
func (p Page) AddForm() *ExpenseForm {
	if p.NewForm != nil {
		return p.NewForm
	}
	return &ExpenseForm{Date: FormatISO(p.Today), DatePreset: "today"}
}

// render writes a page template, reporting a server error in the log only.
func render(w http.ResponseWriter, r *http.Request, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		// The status line is already sent: log it, never show a trace.
		logf("render %s: %v", name, err)
	}
}
