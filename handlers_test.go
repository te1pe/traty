package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// testApp wires a real app around a temporary database and parsed templates.
func testApp(t *testing.T) *app {
	t.Helper()
	if err := parseTemplates(); err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	return &app{store: testStore(t)}
}

// The main scenario end to end: post the form, follow the redirect, see the
// new amount on the dashboard.
func TestCreateExpenseOverHTTP(t *testing.T) {
	a := testApp(t)
	srv := httptest.NewServer(a.routes())
	defer srv.Close()

	catID := mustCategory(t, a.store, "Еда")

	form := url.Values{
		"amount":      {"1250,50"},
		"category":    {itoa(catID)},
		"date_preset": {"today"},
		"comment":     {"Ужин с друзьями"},
		"return_to":   {"/"},
	}
	res, err := http.PostForm(srv.URL+"/expenses", form)
	if err != nil {
		t.Fatalf("POST /expenses: %v", err)
	}
	defer res.Body.Close()

	// Post/Redirect/Get: a refresh must not repost the form.
	if res.Request.URL.Path != "/" {
		t.Errorf("landed on %s, want / after the redirect", res.Request.URL.Path)
	}

	total, count, _ := a.store.SumBetween(Today(), Today())
	if total != 125050 || count != 1 {
		t.Fatalf("today = (%d, %d), want (125050, 1)", total, count)
	}

	body := get(t, srv.URL+"/")
	if !strings.Contains(body, "1 250") || !strings.Contains(body, "Ужин с друзьями") {
		t.Error("the dashboard does not show the new expense")
	}
}

// An invalid amount comes back as the form with the typed values kept.
func TestCreateExpenseValidationKeepsInput(t *testing.T) {
	a := testApp(t)
	srv := httptest.NewServer(a.routes())
	defer srv.Close()

	catID := mustCategory(t, a.store, "Еда")
	res, err := http.PostForm(srv.URL+"/expenses", url.Values{
		"amount":      {"0"},
		"category":    {itoa(catID)},
		"date_preset": {"today"},
		"comment":     {"Ужин"},
	})
	if err != nil {
		t.Fatalf("POST /expenses: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", res.StatusCode)
	}
	body := readAll(t, res)
	if !strings.Contains(body, T(LangRU, "error.amount_zero")) {
		t.Error("the error message is missing")
	}
	if !strings.Contains(body, "Ужин") {
		t.Error("the typed comment was lost")
	}

	if _, count, _ := a.store.SumBetween(Today(), Today()); count != 0 {
		t.Errorf("an invalid expense was stored (%d rows)", count)
	}
}

// Every page must render on an empty database — the first run of a new user.
func TestPagesRenderOnEmptyDatabase(t *testing.T) {
	a := testApp(t)
	srv := httptest.NewServer(a.routes())
	defer srv.Close()

	for _, path := range []string{"/", "/reports", "/categories", "/settings", "/expenses/new"} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, res.StatusCode)
		}
	}
}

// The CSV export contains a header and one row per expense.
func TestCSVExport(t *testing.T) {
	a := testApp(t)
	srv := httptest.NewServer(a.routes())
	defer srv.Close()

	catID := mustCategory(t, a.store, "Еда")
	a.store.CreateExpense(125050, catID, day(2026, 9, 22), "Ужин; с друзьями")

	res, err := http.Get(srv.URL + "/reports.csv?from=2026-09-01&to=2026-09-30")
	if err != nil {
		t.Fatalf("GET /reports.csv: %v", err)
	}
	defer res.Body.Close()

	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition = %q", cd)
	}

	body := readAll(t, res)
	if !strings.HasPrefix(body, "\xef\xbb\xbf") {
		t.Error("the file has no UTF-8 BOM, Excel would mangle Cyrillic")
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want a header and one row", len(lines))
	}
	if !strings.Contains(lines[1], "1250,50") || !strings.Contains(lines[1], "EUR") {
		t.Errorf("row = %q", lines[1])
	}
	// A value containing the separator must be quoted, not split.
	if !strings.Contains(lines[1], `"Ужин; с друзьями"`) {
		t.Errorf("the separator inside a comment is not escaped: %q", lines[1])
	}
}

// return_to must never send the user to another site.
func TestSafeReturn(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/reports", "/reports"},
		{"/?category=3", "/?category=3"},
		{"https://example.com", "/"},
		{"//example.com", "/"},
		{"", "/"},
		{"/ok\r\nX-Injected: 1", "/"},
	}
	for _, c := range cases {
		if got := safeReturn(c.in, "/"); got != c.want {
			t.Errorf("safeReturn(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func get(t *testing.T, url string) string {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer res.Body.Close()
	return readAll(t, res)
}

func readAll(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}
