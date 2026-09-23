package main

import "net/http"

// SettingsData backs /settings.
type SettingsData struct {
	Page
	Settings   Settings
	Currencies []Currency
}

func (a *app) handleSettings(w http.ResponseWriter, r *http.Request) {
	p, err := a.page(w, r, "settings")
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	p.Title = p.T("settings.title")

	s, err := a.store.Settings()
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	render(w, r, http.StatusOK, "settings.html", SettingsData{p, s, Currencies})
}

// handleSettingsSave stores one setting — each option is its own submit button,
// so a choice takes effect the moment it is tapped.
func (a *app) handleSettingsSave(w http.ResponseWriter, r *http.Request) {
	save := func(key, value string, valid bool) {
		if !valid {
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}
		if err := a.store.SetSetting(key, value); err != nil {
			a.serverError(w, r, err)
			return
		}
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
	}

	if v := r.PostFormValue("currency"); v != "" {
		save("currency", v, IsCurrency(v))
		return
	}
	if v := r.PostFormValue("lang"); v != "" {
		save("lang", v, IsLang(v))
		return
	}
	if v := r.PostFormValue("theme"); v != "" {
		save("theme", v, v == "light" || v == "dark" || v == "system")
		return
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}
