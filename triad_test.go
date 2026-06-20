package triad

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestTriad(t *testing.T) {
	r := New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	NotNil(t, r)
	r.Get("/", func(_ http.ResponseWriter, _ *http.Request) error {
		return ErrInternalServerError
	})
	r.ServeHTTP(rec, req)
	Equal(t, http.StatusInternalServerError, rec.Result().StatusCode)
}

func TestCompatHandler(t *testing.T) {
	h := New()
	ts := httptest.NewServer(h)
	defer ts.Close()

	h.Get("/", func(w http.ResponseWriter, _ *http.Request) error {
		return Text(w, "this is test", http.StatusOK)
	})
	h.Get("/compat", Compat(func(w http.ResponseWriter, _ *http.Request) {
		if err := Text(w, "this is compat", http.StatusOK); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	h.Get(
		"/compat/handler",
		CompatHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := Text(w, "this is compat handler", http.StatusOK); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})),
	)

	cases := []struct {
		name string
		req  *http.Request
		want struct {
			code     int
			response string
		}
	}{
		{
			name: "compat-http-handlerfunc",
			req:  must(http.NewRequest(http.MethodGet, ts.URL+"/compat", nil)),
			want: struct {
				code     int
				response string
			}{
				code:     200,
				response: "this is compat",
			},
		},
		{
			name: "compat-http-handler",
			req:  must(http.NewRequest(http.MethodGet, ts.URL+"/compat/handler", nil)),
			want: struct {
				code     int
				response string
			}{
				code:     200,
				response: "this is compat handler",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ts.Client().Do(tt.req)
			NoError(t, err)
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			NoError(t, err)
			Equal(t, tt.want.code, res.StatusCode)
			Equal(t, tt.want.response, string(b))
		})
	}
}

func TestRoutes(t *testing.T) {
	cases := []struct {
		name    string
		req     *http.Request
		pattern string
		status  int
		setup   func(h *Triad)
	}{
		{
			name:    "post",
			pattern: "/post",
			req:     httptest.NewRequest(http.MethodPost, "/post", nil),
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Post("/post", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "get",
			req:     httptest.NewRequest(http.MethodGet, "/get", nil),
			pattern: "/get",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Get("/get", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "connect",
			req:     httptest.NewRequest(http.MethodConnect, "/connect", nil),
			pattern: "/connect",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Connect("/connect", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "put",
			req:     httptest.NewRequest(http.MethodPut, "/put", nil),
			pattern: "/put",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Put("/put", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "delete",
			req:     httptest.NewRequest(http.MethodDelete, "/delete", nil),
			pattern: "/delete",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Delete("/delete", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "patch",
			req:     httptest.NewRequest(http.MethodPatch, "/patch", nil),
			pattern: "/patch",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Patch("/patch", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "head",
			req:     httptest.NewRequest(http.MethodHead, "/head", nil),
			pattern: "/head",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Head("/head", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "trace",
			req:     httptest.NewRequest(http.MethodTrace, "/trace", nil),
			pattern: "/trace",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Trace("/trace", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "options",
			req:     httptest.NewRequest(http.MethodOptions, "/options", nil),
			pattern: "/options",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Options("/options", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
		{
			name:    "custom-method",
			req:     httptest.NewRequest("HELLO", "/hello", nil),
			pattern: "/hello",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Method("HELLO", "/hello", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			tt.setup(r)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, tt.req)
			Equal(t, tt.status, rec.Result().StatusCode)
			Equal(t, "OK", rec.Body.String())

			_, ok := r.RouteInfo().Get(tt.req.Method, tt.req.URL.Path)
			Equal(t, true, ok)
		})
	}
}

func TestCustomMethodFailure(t *testing.T) {
	r := New()
	defer func() { NotNil(t, recover()) }()
	r.Method("", "/", func(_ http.ResponseWriter, _ *http.Request) error { return nil })
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		middleware func(*Triad)
		handler    HandlerFunc
		request    func(ts *httptest.Server) *http.Request
		wantCode   int
		wantResp   string
	}{
		{
			name: "Compatibility",
			middleware: func(r *Triad) {
				r.Use(CompatMiddleware(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Header.Get(HeaderContentType) == MIMEApplicationJSON {
							_ = Text(w, "currently not supported", http.StatusBadRequest)
							return
						}
						next.ServeHTTP(w, r)
					})
				}))
			},
			handler: func(w http.ResponseWriter, _ *http.Request) error {
				return Text(w, "root handler", http.StatusOK)
			},
			request: func(ts *httptest.Server) *http.Request {
				req := must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
				req.Header.Set(HeaderContentType, MIMEApplicationJSON)
				return req
			},
			wantCode: http.StatusBadRequest,
			wantResp: "currently not supported",
		},
		{
			name: "compatibility-calls-next-and-propagates-error",
			middleware: func(r *Triad) {
				r.Use(CompatMiddleware(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						next.ServeHTTP(w, r)
					})
				}))
			},
			handler: func(http.ResponseWriter, *http.Request) error {
				return errors.New("handler error")
			},
			request: func(ts *httptest.Server) *http.Request {
				return must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			},
			wantCode: http.StatusInternalServerError,
			wantResp: `{"error":"handler error","message":"Internal Server Error"}`,
		},
		{
			name: "pass-through",
			middleware: func(r *Triad) {
				r.Use(func(next HandlerFunc) HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) error {
						return next(w, r)
					}
				})
			},
			handler: func(w http.ResponseWriter, _ *http.Request) error {
				return Text(w, "root handler", http.StatusOK)
			},
			request: func(ts *httptest.Server) *http.Request {
				return must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			},
			wantCode: http.StatusOK,
			wantResp: "root handler",
		},
		{
			name: "short-circuit",
			middleware: func(r *Triad) {
				r.Use(func(_ HandlerFunc) HandlerFunc {
					return func(w http.ResponseWriter, _ *http.Request) error {
						return Text(w, "blocked", http.StatusForbidden)
					}
				})
			},
			handler: func(w http.ResponseWriter, _ *http.Request) error {
				return Text(w, "should not execute", http.StatusOK)
			},
			request: func(ts *httptest.Server) *http.Request {
				return must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			},
			wantCode: http.StatusForbidden,
			wantResp: "blocked",
		},
		{
			name: "multiple-middleware",
			middleware: func(r *Triad) {
				r.Use(loggingMiddleware)
				r.Use(authMiddleware)
				r.Use(recoveryMiddleware)
			},
			handler: func(w http.ResponseWriter, _ *http.Request) error {
				return Text(w, "ok", http.StatusOK)
			},
			request: func(ts *httptest.Server) *http.Request {
				return must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			},
			wantCode: http.StatusUnauthorized,
			wantResp: "unauthorized",
		},
		{
			name: "middleware-returns-error",
			middleware: func(r *Triad) {
				r.Use(func(_ HandlerFunc) HandlerFunc {
					return func(_ http.ResponseWriter, _ *http.Request) error {
						return errors.New("middleware error")
					}
				})
			},
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return nil
			},
			request: func(ts *httptest.Server) *http.Request {
				return must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			},
			wantCode: http.StatusInternalServerError,
			wantResp: `{"error":"middleware error","message":"Internal Server Error"}`,
		},
		{
			name:       "handler-returns-error",
			middleware: nil,
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return errors.New("handler error")
			},
			request: func(ts *httptest.Server) *http.Request {
				return must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			},
			wantCode: http.StatusInternalServerError,
			wantResp: `{"error":"handler error","message":"Internal Server Error"}`,
		},
		{
			name: "error-propagation",
			middleware: func(r *Triad) {
				r.Use(func(next HandlerFunc) HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) error {
						err := next(w, r)
						if err != nil {
							return fmt.Errorf("wrapped: %w", err)
						}
						return nil
					}
				})
			},
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return errors.New("handler error")
			},
			request: func(ts *httptest.Server) *http.Request {
				return must(http.NewRequest(http.MethodGet, ts.URL+"/", nil))
			},
			wantCode: http.StatusInternalServerError,
			wantResp: `{"error":"wrapped: handler error","message":"Internal Server Error"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			r.HTTPErrorHandler = DefaultErrHandler(true, r.Logger)
			if tt.middleware != nil {
				tt.middleware(r)
			}
			r.Get("/", tt.handler)
			ts := httptest.NewServer(r)
			defer ts.Close()
			res, err := ts.Client().Do(tt.request(ts))
			NoError(t, err)
			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			NoError(t, err)
			Equal(t, tt.wantCode, res.StatusCode)
			Equal(t, tt.wantResp, string(body))
		})
	}
}

func TestMiddlewareOrder(t *testing.T) {
	var order []string
	r := New()
	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			order = append(order, "mw1-before")
			err := next(w, r)
			order = append(order, "mw1-after")
			return err
		}
	})

	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			order = append(order, "mw2-before")
			err := next(w, r)
			order = append(order, "mw2-after")
			return err
		}
	})

	r.Get("/", func(_ http.ResponseWriter, _ *http.Request) error {
		order = append(order, "handler")
		return nil
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := ts.Client().Get(ts.URL)
	NoError(t, err)

	defer res.Body.Close()
	Equal(t, []string{
		"mw1-before",
		"mw2-before",
		"handler",
		"mw2-after",
		"mw1-after",
	}, order)
}

func TestMiddlewareRegistrationFailure(t *testing.T) {
	r := New()
	defer func() {
		if recov := recover(); recov == nil {
			t.Error("middleware registration after route should fail")
			return
		}
		t.Log("recovered registration of middleware after route")
	}()
	r.Get("/", func(_ http.ResponseWriter, _ *http.Request) error { return nil })
	r.Use(func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) error {
			return next(w, r)
		}
	})
}

func TestStart(t *testing.T) {
	r := New()
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) error {
		return Text(w, "OK", http.StatusTeapot)
	})
	port, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer port.Close()
	done := make(chan error, 1)
	go func() {
		done <- r.Start(port.Addr().String())
	}()

	select {
	case <-time.After(250 * time.Millisecond):
		t.Fatal("start did not error out")
	case err := <-done:
		var opErr *net.OpError
		if !errors.As(err, &opErr) {
			t.Fatalf("expected net.OpError, got %v", err)
		}
	}
}

type ctxKey struct{ key string }

func TestWith(t *testing.T) {
	var mwi1, hl1, mwi2, hl2 uint8

	mw1 := func(next HandlerFunc) HandlerFunc {
		mwi1++
		return func(w http.ResponseWriter, r *http.Request) error {
			hl1++
			r = r.WithContext(context.WithValue(r.Context(), ctxKey{"inline1"}, "yes"))
			return next(w, r)
		}
	}
	mw2 := func(next HandlerFunc) HandlerFunc {
		mwi2++
		return func(w http.ResponseWriter, r *http.Request) error {
			hl2++
			r = r.WithContext(context.WithValue(r.Context(), ctxKey{"inline2"}, "yes"))
			return next(w, r)
		}
	}

	r := New()
	r.Get("/hi", func(w http.ResponseWriter, _ *http.Request) error {
		_, err := w.Write([]byte("bye"))
		return err
	})
	r.With(mw1).With(mw2).Get("/inline", func(w http.ResponseWriter, r *http.Request) error {
		v1, ok := r.Context().Value(ctxKey{"inline1"}).(string)
		if !ok {
			v1 = "no"
		}
		v2, ok := r.Context().Value(ctxKey{"inline2"}).(string)
		if !ok {
			v2 = "no"
		}
		_, err := fmt.Fprintf(w, "inline %s %s", v1, v2)
		return err
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	cases := []struct {
		name    string
		method  string
		pattern string
		want    string
	}{
		{
			name:    "/hi",
			method:  http.MethodGet,
			pattern: "/hi",
			want:    "bye",
		},
		{
			name:    "/inline",
			method:  "GET",
			pattern: "/inline",
			want:    "inline yes yes",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, ts.URL+tt.pattern, nil)
			NoError(t, err)
			res, err := ts.Client().Do(req)
			NoError(t, err)
			defer res.Body.Close()
			b, err := io.ReadAll(res.Body)
			NoError(t, err)
			Equal(t, tt.want, string(b))
		})
	}

	var want uint8 = 1
	Equal(t, want, mwi1)
	Equal(t, want, hl1)
	Equal(t, want, mwi2)
	Equal(t, want, hl2)
}

func TestGroup(t *testing.T) {
	tests := []struct {
		name       string
		parent     string
		pattern    string
		callback   bool
		wantPrefix string
	}{
		{
			name:       "simple",
			parent:     "",
			pattern:    "/api",
			wantPrefix: "/api",
		},
		{
			name:       "nested",
			parent:     "/api",
			pattern:    "/v1",
			wantPrefix: "/api/v1",
		},
		{
			name:       "callback",
			parent:     "/api",
			pattern:    "/v1",
			callback:   true,
			wantPrefix: "/api/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false

			tr := &Triad{
				prefix: tt.parent,
			}

			got := tr.Group(tt.pattern, func(_ *Triad) {
				called = true
			})

			if got.prefix != tt.wantPrefix {
				t.Fatalf("prefix = %q want %q",
					got.prefix, tt.wantPrefix)
			}

			if tt.callback && !called {
				t.Fatal("callback not executed")
			}
		})
	}
}

func NotNil(t testing.TB, v any) {
	t.Helper()
	if v == nil {
		t.Errorf("%v is nil", v)
	}
}

func Equal(t testing.TB, e, v any) {
	t.Helper()
	if !reflect.DeepEqual(e, v) {
		t.Errorf("expected: %v, got: %v", e, v)
	}
}

func Nil(t testing.TB, v any) {
	t.Helper()
	if v != nil {
		t.Errorf("expected nil value instead got %v", v)
	}
}

func NoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("expected nil error got %s", err.Error())
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func authMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			return Text(w, "unauthorized", http.StatusUnauthorized)
		}
		return next(w, r)
	}
}

func loggingMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		return next(w, r)
	}
}

func recoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				err = Text(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		return next(w, r)
	}
}
