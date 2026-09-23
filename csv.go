package main

import (
	"encoding/csv"
	"net/http"
	"strings"
)

// csvBOM makes Excel read the UTF-8 file as UTF-8 instead of mangling Cyrillic.
const csvBOM = "\xef\xbb\xbf"

// handleReportsCSV streams the expenses of a period as a CSV download.
func (a *app) handleReportsCSV(w http.ResponseWriter, r *http.Request) {
	settings, err := a.store.Settings()
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	today := Today()
	_, from, to, rangeErr := reportRange(r.URL.Query(), today)
	if rangeErr {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}

	items, err := a.store.ExpensesBetweenAsc(from, to)
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	filename := "traty_" + FormatISO(from) + "_" + FormatISO(to) + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	if _, err := w.Write([]byte(csvBOM)); err != nil {
		logf("csv write: %v", err)
		return
	}

	cw := csv.NewWriter(w)
	cw.Comma = ';' // opens by double-click in Excel with a comma decimal mark
	lang := settings.Lang

	header := []string{
		T(lang, "csv.date"), T(lang, "csv.category"), T(lang, "csv.amount"),
		T(lang, "csv.currency"), T(lang, "csv.comment"),
	}
	if err := cw.Write(header); err != nil {
		logf("csv write: %v", err)
		return
	}

	for _, e := range items {
		row := []string{
			FormatISO(e.Date),
			csvSafe(e.CategoryName),
			FormatCSVAmount(e.AmountMinor),
			settings.Currency,
			csvSafe(e.Comment),
		}
		if err := cw.Write(row); err != nil {
			logf("csv write: %v", err)
			return
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		logf("csv flush: %v", err)
	}
}

// csvSafe defuses spreadsheet formula injection: a value starting with one of
// these characters is prefixed with an apostrophe so Excel treats it as text.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	if strings.ContainsAny(s[:1], "=+-@\t\r") {
		return "'" + s
	}
	return s
}
