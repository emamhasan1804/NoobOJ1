package handlers

import (
	"NoobOJ/database"
	"net/http"
)

func RequiredLogin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if session.Values["username"] == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func RequiredAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "session")
		if session.Values["username"] == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		var usr string
		database.DB.QueryRow("SELECT user_type FROM users WHERE username=?", session.Values["username"]).Scan(&usr)
		if usr != "admin" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
