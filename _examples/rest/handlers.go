package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/jshk00/triad"
)

type ArticleHandler struct {
	db *DB
}

func (a *ArticleHandler) List(w http.ResponseWriter, r *http.Request) error {
	tag := r.URL.Query().Get("tag")
	if tag != "" {
		list := a.db.ListByTag(tag)
		return triad.JSON(w, list, http.StatusOK)
	}
	list := a.db.List()
	return triad.JSON(w, list, http.StatusOK)
}

func (a *ArticleHandler) Get(w http.ResponseWriter, r *http.Request) error {
	id, err := triad.ParseParam[int](r, "id")
	if err != nil {
		return triad.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	article := a.db.Get(id)
	if article == nil {
		return triad.NewHTTPError(http.StatusNotFound)
	}
	return triad.JSON(w, article, http.StatusOK)
}

func (a *ArticleHandler) Delete(w http.ResponseWriter, r *http.Request) error {
	id, err := triad.ParseParam[int](r, "id")
	if err != nil {
		return triad.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	a.db.Delete(id)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *ArticleHandler) Post(w http.ResponseWriter, r *http.Request) error {
	article := &Article{}
	if err := triad.Bind(r, article); err != nil {
		return err
	}
	a.db.Save(article)
	w.WriteHeader(http.StatusCreated)
	return nil
}

func (a *ArticleHandler) Put(w http.ResponseWriter, r *http.Request) error {
	var article *Article
	if err := triad.Bind(r, article); err != nil {
		return err
	}
	a.db.Update(article)
	return nil
}

func AdminOnly(next triad.HandlerFunc) triad.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		isAdmin, ok := r.Context().Value("acl.admin").(bool)
		if !ok || !isAdmin {
			return triad.NewHTTPError(http.StatusForbidden, http.StatusText(http.StatusForbidden)).
				WithText()
		}
		return next(w, r)
	}
}

func Auth(next triad.HandlerFunc) triad.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		auth := r.Header.Get("Authorization")

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && parts[0] == "Basic" {
			b, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				return triad.NewHTTPError(http.StatusUnauthorized, "invalid base64")
			}
			decoded := strings.SplitN(string(b), ":", 2)
			if len(decoded) == 2 {
				username := decoded[0]
				password := decoded[1]
				if username == "admin" && password == "admin" {
					r = r.WithContext(context.WithValue(r.Context(), "acl.admin", true))
				}
			}
		}
		return next(w, r)
	}
}

func AdminRouter(h *triad.Triad) {
	h.Use(Auth, AdminOnly)
	h.Get("/", func(w http.ResponseWriter, r *http.Request) error {
		_, err := w.Write([]byte("admin: index"))
		return err
	})
	h.Get("/accounts", func(w http.ResponseWriter, r *http.Request) error {
		_, err := w.Write([]byte("admin: list accounts.."))
		return err
	})
	h.Get("/users/{userId}", func(w http.ResponseWriter, r *http.Request) error {
		_, err := w.Write([]byte(fmt.Sprintf("admin: view user id %v", r.PathValue("userId"))))
		return err
	})
}
