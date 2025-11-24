package middleware

import (
        "net/http"

        "github.com/gorilla/sessions"
)

func RequireAuth(store *sessions.CookieStore) func(http.Handler) http.Handler {
        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        session, _ := store.Get(r, "session")
                        userID := session.Values["user_id"]

                        if userID == nil {
                                http.Redirect(w, r, "/login", http.StatusSeeOther)
                                return
                        }

                        next.ServeHTTP(w, r)
                })
        }
}

func RequireAdmin(store *sessions.CookieStore) func(http.Handler) http.Handler {
        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        session, _ := store.Get(r, "session")
                        userID := session.Values["user_id"]
                        userRole := session.Values["role"]

                        if userID == nil {
                                http.Redirect(w, r, "/login", http.StatusSeeOther)
                                return
                        }

                        if userRole != "admin" {
                                http.Error(w, "Доступ запрещен", http.StatusForbidden)
                                return
                        }

                        next.ServeHTTP(w, r)
                })
        }
}
