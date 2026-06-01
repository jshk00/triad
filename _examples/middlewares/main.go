package main

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jshk00/triad"
)

func main() {
	h := triad.New()
	h.Use(Logger)
	h.With(RequestID).Get("/reqID", func(w http.ResponseWriter, r *http.Request) error {
		return triad.Text(w, r.Context().Value(ctxReqIDKey).(string), http.StatusOK)
	})
	h.Group("/admin", func(h *triad.Triad) {
		h.Use(
			RequestID,
			Auth,
			AdminOnly,
			// middleware registerd with compatibility
			triad.CompatMiddleware(AllowContentType(triad.MIMETextPlain)),
		)
		h.Get("/", func(w http.ResponseWriter, r *http.Request) error {
			return triad.Text(w, "admin root", http.StatusOK)
		})
	})

	if err := r.Start(":8080"); err != nil {
		panic(err)
	}
}

func Logger(next triad.HandlerFunc) triad.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		start := time.Now()
		rw := &respWriter{ResponseWriter: w}
		err := next(rw, r)
		if err != nil {
			if he, ok := errors.AsType[*triad.HTTPError](err); ok {
				rw.code = he.StatusCode
			}
		}
		end := time.Since(start)
		slog.Info(
			"request received",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_address", r.RemoteAddr),
			slog.Int("status_code", rw.code),
			slog.String("latency", end.String()),
		)
		return err
	}
}

type respWriter struct {
	http.ResponseWriter
	code int
}

func (w *respWriter) WriteHeader(statusCode int) {
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func RequestID(next triad.HandlerFunc) triad.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		r = r.WithContext(context.WithValue(r.Context(), ctxReqIDKey, idgen.next()))
		return next(w, r)
	}
}

const ctxReqIDKey = "request.id"

var idgen = &idGen{}

type idGen struct {
	mu      sync.Mutex
	current int
}

func (ig *idGen) next() string {
	var s string
	ig.mu.Lock()
	ig.current++
	s = time.Now().Format(time.RFC3339) + "-" + strconv.Itoa(ig.current)
	ig.mu.Unlock()
	return s
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

// AllowContentType enforces a whitelist of request Content-Types otherwise responds
// with a 415 Unsupported Media Type status.
func AllowContentType(contentTypes ...string) func(http.Handler) http.Handler {
	allowedContentTypes := make(map[string]struct{}, len(contentTypes))
	for _, ctype := range contentTypes {
		allowedContentTypes[strings.TrimSpace(strings.ToLower(ctype))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength == 0 {
				// Skip check for empty content body
				next.ServeHTTP(w, r)
				return
			}

			s, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
			s = strings.ToLower(strings.TrimSpace(s))

			if _, ok := allowedContentTypes[s]; ok {
				next.ServeHTTP(w, r)
				return
			}

			w.WriteHeader(http.StatusUnsupportedMediaType)
		})
	}
}
