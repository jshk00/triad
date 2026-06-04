package triad

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestHypr(t *testing.T) {
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

func TestCompat(t *testing.T) {
	h := New()
	ts := httptest.NewServer(h)
	defer ts.Close()

	h.Use(CompatMiddleware(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(HeaderContentType) == MIMEApplicationJSON {
				if err := Text(w, "currently not supported", http.StatusBadRequest); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
			h.ServeHTTP(w, r)
		})
	}))
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
		{
			name: "compat-http-middleware",
			req: must(func() (*http.Request, error) {
				req, err := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
				if err != nil {
					return nil, err
				}
				req.Header.Set(HeaderContentType, MIMEApplicationJSON)
				return req, nil
			}()),
			want: struct {
				code     int
				response string
			}{
				code:     400,
				response: "currently not supported",
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
			name:    "hypr-post",
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
			name:    "hypr-get",
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
			name:    "hypr-put",
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
			name:    "hypr-delete",
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
			name:    "hypr-patch",
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
			name:    "hypr-head",
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
			name:    "hypr-trace",
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
			name:    "hypr-options",
			req:     httptest.NewRequest(http.MethodOptions, "/options", nil),
			pattern: "/options",
			status:  http.StatusTeapot,
			setup: func(h *Triad) {
				h.Options("/options", func(w http.ResponseWriter, _ *http.Request) error {
					return Text(w, "OK", http.StatusTeapot)
				})
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			tt.setup(h)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, tt.req)
			_, ok := h.routes.Get(tt.req.Method, tt.pattern)
			Equal(t, true, ok)
			Equal(t, tt.status, rec.Result().StatusCode)
			Equal(t, "OK", rec.Body.String())
		})
	}
}

func TestMiddleware(t *testing.T)         {}
func TestStart(t *testing.T)              {}
func TestGroup(t *testing.T)              {}
func TestMoung(t *testing.T)              {}
func TestErrorHandler(t *testing.T)       {}
func TestCustomErrorHandler(t *testing.T) {}

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
