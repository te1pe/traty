package main

import (
	"net/http"
	"strings"
)

// CategoryRow is a category plus how many expenses use it, so the list can
// explain why one cannot be deleted.
type CategoryRow struct {
	Category
	Count int
}

// CategoriesData backs /categories.
type CategoriesData struct {
	Page
	Rows      []CategoryRow
	NewName   string
	Error     string
	BlockedID int64
}

func (a *app) handleCategories(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "categories")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	p.Title = p.T("categories.title")

	data, err := a.categoriesData(p)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	render(w, r, http.StatusOK, "categories.html", data)
}

func (a *app) categoriesData(p Page) (*CategoriesData, error) {
	rows := make([]CategoryRow, 0, len(p.Categories))
	for _, c := range p.Categories {
		n, err := a.store.CategoryExpenseCount(c.ID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, CategoryRow{Category: c, Count: n})
	}
	return &CategoriesData{Page: p, Rows: rows}, nil
}

// handleCategoryCreate adds a category from the form at the top of the page.
func (a *app) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "categories")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	p.Title = p.T("categories.title")

	name := strings.TrimSpace(r.PostFormValue("name"))
	showError := func(msg string) {
		data, derr := a.categoriesData(p)
		if derr != nil {
			a.serverError(w, r, derr)
			return
		}
		data.NewName = name
		data.Error = msg
		render(w, r, http.StatusUnprocessableEntity, "categories.html", data)
	}

	switch {
	case name == "":
		showError(p.T("error.category_name"))
		return
	case len([]rune(name)) > maxCategoryNameLen:
		showError(p.T("error.name_long"))
		return
	}

	id, err := a.store.CreateCategory(name)
	if err == ErrDuplicateCategory {
		showError(p.T("error.category_dup"))
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	setFlash(w, "category_added", id)
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

// handleCategoryDelete removes a category that no expense uses. A category in
// use is kept and the page explains why.
func (a *app) handleCategoryDelete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.PathValue("id"))

	err := a.store.DeleteCategory(id)
	if err == ErrCategoryInUse {
		p, perr := a.page(w, r, "categories")
		if perr != nil {
			a.serverError(w, r, perr)
			return
		}
		p.Title = p.T("categories.title")

		data, derr := a.categoriesData(p)
		if derr != nil {
			a.serverError(w, r, derr)
			return
		}
		for _, row := range data.Rows {
			if row.ID == id {
				data.Error = p.Tf("categories.in_use", row.Count)
				data.BlockedID = id
			}
		}
		render(w, r, http.StatusConflict, "categories.html", data)
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	setFlash(w, "category_deleted", 0)
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}
