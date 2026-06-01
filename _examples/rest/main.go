package main

import (
	"context"
	"net/http"

	"github.com/jshk00/triad"
)

func main() {
	r := triad.New()
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) error {
		return triad.Text(w, "pong", http.StatusOK)
	})
	db := &DB{
		data:  make([]*Article, 0),
		index: make(map[int]int),
	}
	articleHandler := &ArticleHandler{db}
	// Grouping routes for articles
	r.Group("/articles", func(r *triad.Triad) {
		r.Get("/{id}", articleHandler.Get)
		r.Get("", articleHandler.List)
		r.Post("", articleHandler.Post)
		r.Put("", articleHandler.Put)
		r.Delete("/{id}", articleHandler.Delete)
	})

	// Mounting Different router
	r.Group("/admin", AdminRouter)

	if err := (&triad.Server{Address: ":8080"}).server.Start(context.Background(), r); err != nil {
		panic(err)
	}
}
