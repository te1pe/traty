package main

import (
	"database/sql"
	"errors"
	"net/http"
)

// ExpensePageData backs /expenses/new — the full-page form used as a fallback
// and whenever a submitted form comes back with errors.
type ExpensePageData struct {
	Page
	Form *ExpenseForm
}

// handleExpenseNew shows the expense form as its own page. The sheet in the
// tab bar is a popover and needs no request, so this is the fallback path.
func (a *app) handleExpenseNew(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "overview")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	p.Title = p.T("form.new_title")

	form := &ExpenseForm{
		Date:       FormatISO(p.Today),
		DatePreset: "today",
		ReturnTo:   safeReturn(r.URL.Query().Get("return_to"), "/"),
		Errors:     map[string]string{},
	}
	render(w, r, http.StatusOK, "expense_page.html", ExpensePageData{p, form})
}

// handleExpenseCreate stores a new expense (Post/Redirect/Get). On a validation
// error it re-renders the form page with the values the user typed.
func (a *app) handleExpenseCreate(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "overview")
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	form := expenseFormFrom(r)
	amount, categoryID, date, ok := form.validated(a.store, p.Lang, p.Today)
	if !ok {
		p.Title = p.T("form.new_title")
		render(w, r, http.StatusUnprocessableEntity, "expense_page.html", ExpensePageData{p, form})
		return
	}

	id, err := a.store.CreateExpense(amount, categoryID, date, form.Comment)
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	setFlash(w, "expense_added", id)
	http.Redirect(w, r, form.ReturnTo, http.StatusSeeOther)
}

// handleExpenseEdit draws the dashboard with the edit sheet open over it.
func (a *app) handleExpenseEdit(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	p, err := a.page(w, r, "overview")
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	expense, err := a.store.Expense(id)
	if errors.Is(err, sql.ErrNoRows) {
		a.handleNotFound(w, r)
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	p.Title = p.T("form.edit_title")
	data, err := a.overview(p)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	data.Edit = expenseFormOf(expense, p.Today, safeReturn(r.URL.Query().Get("return_to"), "/"))
	render(w, r, http.StatusOK, "overview.html", data)
}

// handleExpenseUpdate saves an edited expense.
func (a *app) handleExpenseUpdate(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	p, err := a.page(w, r, "overview")
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	form := expenseFormFrom(r)
	form.ID = id

	amount, categoryID, date, ok := form.validated(a.store, p.Lang, p.Today)
	if !ok {
		p.Title = p.T("form.edit_title")
		render(w, r, http.StatusUnprocessableEntity, "expense_page.html", ExpensePageData{p, form})
		return
	}

	err = a.store.UpdateExpense(id, amount, categoryID, date, form.Comment)
	if errors.Is(err, sql.ErrNoRows) {
		a.handleNotFound(w, r)
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	setFlash(w, "expense_saved", id)
	http.Redirect(w, r, form.ReturnTo, http.StatusSeeOther)
}

// handleExpenseDelete removes one expense after the confirmation dialog.
func (a *app) handleExpenseDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))
	returnTo := safeReturn(r.PostFormValue("return_to"), "/")

	err := a.store.DeleteExpense(id)
	if errors.Is(err, sql.ErrNoRows) {
		a.handleNotFound(w, r)
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	setFlash(w, "expense_deleted", 0)
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}
