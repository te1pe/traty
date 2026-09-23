// Command traty is a small personal expense tracker: one Go binary, one SQLite
// file, server-rendered HTML. Run it with `go run .` and open localhost:8080.
package main

import (
	"embed"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed static
var staticFS embed.FS

// app holds the dependencies every handler needs.
type app struct {
	store *Store
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("traty: %v", err)
	}
}

func run() error {
	port := envOr("PORT", "8080")
	dbPath := envOr("DATABASE_PATH", "data/finance.db")

	if err := parseTemplates(); err != nil {
		return err
	}

	store, err := OpenStore(dbPath)
	if err != nil {
		return err
	}
	defer store.Close()

	a := &app{store: store}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      a.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Траты: база %s", dbPath)
	log.Printf("  на этом компьютере: http://localhost:%s", port)
	if ip := localIP(); ip != "" {
		log.Printf("  с телефона в той же сети Wi-Fi: http://%s:%s", ip, port)
	}
	return srv.ListenAndServe()
}

// routes wires every URL. The patterns use the method-aware routing of
// net/http, so no router library is needed.
func (a *app) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", a.handleOverview)
	mux.HandleFunc("GET /expenses/new", a.handleExpenseNew)
	mux.HandleFunc("POST /expenses", a.handleExpenseCreate)
	mux.HandleFunc("GET /expenses/{id}/edit", a.handleExpenseEdit)
	mux.HandleFunc("POST /expenses/{id}", a.handleExpenseUpdate)
	mux.HandleFunc("POST /expenses/{id}/delete", a.handleExpenseDelete)

	mux.HandleFunc("GET /reports", a.handleReports)
	mux.HandleFunc("GET /reports.csv", a.handleReportsCSV)

	mux.HandleFunc("GET /categories", a.handleCategories)
	mux.HandleFunc("POST /categories", a.handleCategoryCreate)
	mux.HandleFunc("POST /categories/{id}/delete", a.handleCategoryDelete)

	mux.HandleFunc("GET /settings", a.handleSettings)
	mux.HandleFunc("POST /settings", a.handleSettingsSave)

	// The manifest needs its own media type so the browser offers "add to
	// home screen"; everything else under /static is served as-is.
	mux.HandleFunc("GET /static/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		http.ServeFileFS(w, r, staticFS, "static/manifest.webmanifest")
	})
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	// Anything else is a 404 rendered in the app's own design.
	mux.HandleFunc("/", a.handleNotFound)

	return mux
}

// ---------- shared request helpers ----------

// page builds the common view data for a request: settings, navigation state,
// the categories the "add expense" sheet needs, and any pending toast.
func (a *app) page(w http.ResponseWriter, r *http.Request, nav string) (Page, error) {
	s, err := a.store.Settings()
	if err != nil {
		return Page{}, err
	}
	cats, err := a.store.Categories()
	if err != nil {
		return Page{}, err
	}

	p := Page{
		Lang:       s.Lang,
		Theme:      s.Theme,
		Currency:   s.Currency,
		Nav:        nav,
		Path:       currentPath(r),
		Today:      Today(),
		Categories: cats,
	}
	if key, id, ok := takeFlash(w, r); ok {
		p.Toast = T(p.Lang, "toast."+key)
		p.NewID = id
	}
	return p, nil
}

// flashCookie carries a one-shot toast across the redirect of a POST.
const flashCookie = "flash"

// setFlash stores "<key>:<id>" until the next page render.
func setFlash(w http.ResponseWriter, key string, id int64) {
	value := key
	if id > 0 {
		value = key + ":" + itoa(id)
	}
	http.SetCookie(w, &http.Cookie{
		Name: flashCookie, Value: value, Path: "/",
		MaxAge: 60, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

// takeFlash reads and clears the pending toast.
func takeFlash(w http.ResponseWriter, r *http.Request) (key string, id int64, ok bool) {
	c, err := r.Cookie(flashCookie)
	if err != nil || c.Value == "" {
		return "", 0, false
	}
	http.SetCookie(w, &http.Cookie{Name: flashCookie, Value: "", Path: "/", MaxAge: -1})

	key, rest, _ := strings.Cut(c.Value, ":")
	if !isFlashKey(key) {
		return "", 0, false
	}
	return key, parseID(rest), true
}

// isFlashKey keeps the cookie from selecting arbitrary translation keys.
func isFlashKey(k string) bool {
	switch k {
	case "expense_added", "expense_saved", "expense_deleted", "category_added", "category_deleted":
		return true
	}
	return false
}

// safeReturn validates a return_to value: only internal paths are accepted,
// so a form can never redirect the user off-site.
func safeReturn(v, fallback string) string {
	if v == "" || !strings.HasPrefix(v, "/") || strings.HasPrefix(v, "//") {
		return fallback
	}
	if strings.ContainsAny(v, "\r\n") {
		return fallback
	}
	return v
}

// serverError logs the real cause and shows the user a plain message.
func (a *app) serverError(w http.ResponseWriter, r *http.Request, err error) {
	logf("%s %s: %v", r.Method, r.URL.Path, err)

	lang := LangRU
	if s, e := a.store.Settings(); e == nil {
		lang = s.Lang
	}
	p := Page{Lang: lang, Theme: "system", Today: Today(), Title: T(lang, "error.server")}
	render(w, r, http.StatusInternalServerError, "error.html",
		struct {
			Page
			Message string
		}{p, T(lang, "error.server")})
}

func (a *app) handleNotFound(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	p.Title = p.T("error.not_found")
	render(w, r, http.StatusNotFound, "error.html",
		struct {
			Page
			Message string
		}{p, p.T("error.not_found")})
}

// localIP returns this machine's address in the local network, so a phone on
// the same Wi-Fi can be pointed at the app. Empty when there is no such address.
func localIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

// currentPath is the path (with query) a form returns to after saving.
func currentPath(r *http.Request) string {
	p := r.URL.Path
	if r.URL.RawQuery != "" {
		p += "?" + r.URL.RawQuery
	}
	return safeReturn(p, "/")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func logf(format string, args ...any) { log.Printf(format, args...) }
