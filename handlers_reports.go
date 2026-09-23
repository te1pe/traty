package main

import (
	"net/http"
	"net/url"
	"time"
)

// reportListLimit is how many expenses a report shows before the "show more"
// link; the CSV always contains every expense of the period.
const reportListLimit = 50

// DayGroup is one day of the report list.
type DayGroup struct {
	Date  time.Time
	Total int64
	Items []Expense
}

// ReportData backs /reports.
type ReportData struct {
	Page
	Period     string // week | month | prev | year | custom
	From       time.Time
	To         time.Time
	Total      int64
	Count      int
	MaxTotal   int64
	ByCategory []CategoryTotal
	Days       []DayGroup
	Shown      int
	Hidden     int
	RangeError string
	CSVQuery   string
}

// reportRange resolves the requested period into two dates. Presets keep the
// URLs short; explicit from/to always win.
func reportRange(q url.Values, today time.Time) (period string, from, to time.Time, rangeErr bool) {
	if qf, qt := q.Get("from"), q.Get("to"); qf != "" || qt != "" {
		f, errF := ParseDate(qf)
		t, errT := ParseDate(qt)
		if errF != nil {
			f = MonthStart(today)
		}
		if errT != nil {
			t = today
		}
		if t.Before(f) {
			return "custom", f, t, true
		}
		return "custom", f, t, false
	}

	switch q.Get("period") {
	case "week":
		return "week", WeekStart(today), today, false
	case "prev":
		prev := MonthStart(today).AddDate(0, -1, 0)
		return "prev", prev, MonthStart(today).AddDate(0, 0, -1), false
	case "year":
		return "year", time.Date(today.Year(), 1, 1, 0, 0, 0, 0, today.Location()), today, false
	default:
		return "month", MonthStart(today), today, false
	}
}

func (a *app) handleReports(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "reports")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	p.Title = p.T("reports.title")

	q := r.URL.Query()
	period, from, to, rangeErr := reportRange(q, p.Today)

	data := &ReportData{Page: p, Period: period, From: from, To: to}
	if rangeErr {
		data.RangeError = p.T("error.range_order")
		render(w, r, http.StatusOK, "reports.html", data)
		return
	}

	total, count, err := a.store.SumBetween(from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	totals, err := a.store.TotalsByCategory(from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	items, err := a.store.ExpensesBetween(from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	shown := items
	if q.Get("all") != "1" && len(items) > reportListLimit {
		shown = items[:reportListLimit]
		data.Hidden = len(items) - reportListLimit
	}

	data.Total = total
	data.Count = count
	data.ByCategory = WithPercent(totals, total)
	data.MaxTotal = MaxTotal(totals)
	data.Days = groupByDay(shown)
	data.Shown = len(shown)
	data.CSVQuery = "from=" + FormatISO(from) + "&to=" + FormatISO(to)

	render(w, r, http.StatusOK, "reports.html", data)
}

// groupByDay splits a date-ordered list of expenses into day groups.
func groupByDay(items []Expense) []DayGroup {
	var out []DayGroup
	for _, e := range items {
		if n := len(out); n > 0 && out[n-1].Date.Equal(e.Date) {
			out[n-1].Items = append(out[n-1].Items, e)
			out[n-1].Total += e.AmountMinor
			continue
		}
		out = append(out, DayGroup{Date: e.Date, Total: e.AmountMinor, Items: []Expense{e}})
	}
	return out
}
