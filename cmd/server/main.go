package main

import (
	"log"
	"net/http"
	"os"
	"petropavlovsk-budget/internal/db"
	"petropavlovsk-budget/internal/handlers"
	"petropavlovsk-budget/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
)

func main() {
	database, err := db.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "default-secret-key-change-in-production"
	}

	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}

	h := handlers.New(database, store)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			next.ServeHTTP(w, r)
		})
	})

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	r.Get("/", h.Home)
	r.Get("/register", h.RegisterPage)
	r.Post("/register", h.RegisterSubmit)
	r.Get("/login", h.LoginPage)
	r.Post("/login", h.LoginSubmit)
	r.Get("/logout", h.Logout)

	r.Get("/projects", h.ProjectsPage)
	r.Get("/projects/{id}", h.ProjectDetail)
	r.Get("/map", h.MapPage)
	r.Get("/api/map/data", h.MapData)
	r.Get("/api/map/popup/{id}", h.ProjectPopup)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(store))
		r.Get("/submit", h.SubmitPage)
		r.Post("/submit", h.SubmitProject)
		r.Post("/vote", h.VoteSubmit)
	})

	log.Println("Server starting on http://0.0.0.0:5000")
	if err := http.ListenAndServe("0.0.0.0:5000", r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
