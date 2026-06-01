package main

import (
	"log/slog"
	"net/http"

	"github.com/jshk00/triad"
)

func main() {
	r := triad.New()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
		return triad.Text(w, "root", http.StatusOK)
	})
	// internal routing group
	r.Group("/admin", func(r *triad.Triad) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) error {
			return triad.Text(w, "admin root", http.StatusOK)
		})
	})
	// external routing group simulated function of mounted router
	r.Group("/book", bookRouter)
	if err := r.Start(":8080"); err != nil {
		panic(err)
	}
}

func bookRouter(r *triad.Triad) {
	r.Use(func(next triad.HandlerFunc) triad.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			slog.Info("executed middleware")
			return next(w, r)
		}
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) error {
		id, err := triad.ParseParam[int](r, "id")
		if err != nil {
			return err
		}
		return triad.JSON(
			w,
			map[string]any{"id": id, "title": "the book of secrets"},
			http.StatusOK,
		)
	})
}
