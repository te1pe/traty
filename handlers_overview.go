package main

import (
	"database/sql"
	"errors"
	"net/http"
	"time"
)

// OverviewData is the dashboard: today, this week with the donut, and the last
// five expenses. The same page also draws the overlays (an expense being
// edited, or a category's details) so those keep their own URL.
type OverviewData struct {
	Page
	TodayTotal int64
	TodayCount int
	WeekTotal  int64
	WeekCount  int
	WeekFrom   time.Time
	WeekTo     time.Time
	Donut      []CategoryTotal
	Recent     []Expense
	HasAny     bool // any expense ever recorded
	Edit       *ExpenseForm
	Detail     *CategoryDetail
}

// CategoryDetail is the "what is inside this category" overlay.
type CategoryDetail struct {
	Category Category
	Period   string // week | month
	From     time.Time
	To       time.Time
	Total    int64
	Count    int
	Percent  float64
	Expenses []Expense
}

// overview gathers the dashboard numbers. The week runs Monday → today,
// not the last seven days.
func (a *app) overview(p Page) (*OverviewData, error) {
	today := p.Today
	weekFrom := WeekStart(today)

	todayTotal, todayCount, err := a.store.SumBetween(today, today)
	if err != nil {
		return nil, err
	}
	weekTotal, weekCount, err := a.store.SumBetween(weekFrom, today)
	if err != nil {
		return nil, err
	}
	totals, err := a.store.TotalsByCategory(weekFrom, today)
	if err != nil {
		return nil, err
	}
	recent, err := a.store.RecentExpenses(5)
	if err != nil {
		return nil, err
	}

	return &OverviewData{
		Page:       p,
		TodayTotal: todayTotal,
		TodayCount: todayCount,
		WeekTotal:  weekTotal,
		WeekCount:  weekCount,
		WeekFrom:   weekFrom,
		WeekTo:     today,
		Donut:      BuildDonut(totals, weekTotal, T(p.Lang, "donut.other")),
		Recent:     recent,
		HasAny:     len(recent) > 0,
	}, nil
}

func (a *app) handleOverview(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "overview")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	p.Title = p.T("nav.overview")

	data, err := a.overview(p)
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	// ?category=<id> opens the category overlay over the dashboard.
	if id := parseID(r.URL.Query().Get("category")); id > 0 {
		detail, err := a.categoryDetail(p, id, r.URL.Query().Get("period"))
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if err != nil {
			a.serverError(w, r, err)
			return
		}
		data.Detail = detail
		data.Page.ActiveCategory = id
	}

	render(w, r, http.StatusOK, "overview.html", data)
}

// categoryDetail collects one category's expenses for the chosen period.
func (a *app) categoryDetail(p Page, id int64, period string) (*CategoryDetail, error) {
	cat, err := a.store.Category(id)
	if err != nil {
		return nil, err
	}

	from := WeekStart(p.Today)
	if period == "month" {
		from = MonthStart(p.Today)
	} else {
		period = "week"
	}

	items, err := a.store.CategoryExpensesBetween(id, from, p.Today)
	if err != nil {
		return nil, err
	}
	periodTotal, _, err := a.store.SumBetween(from, p.Today)
	if err != nil {
		return nil, err
	}

	var total int64
	for _, e := range items {
		total += e.AmountMinor
	}
	var percent float64
	if periodTotal > 0 {
		percent = float64(total) / float64(periodTotal) * 100
	}

	return &CategoryDetail{
		Category: cat, Period: period, From: from, To: p.Today,
		Total: total, Count: len(items), Percent: percent, Expenses: items,
	}, nil
}
