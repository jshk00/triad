package triad

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultHTTPHandler(t *testing.T) {
	cases := []struct {
		name       string
		handler    HandlerFunc
		errHandler HTTPErrorHandler
		wantCode   int
		wantResp   string
	}{
		{
			name: "success",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return errors.New("handler failed")
			},
			wantCode: http.StatusInternalServerError,
			wantResp: `{"message":"Internal Server Error"}`,
		},
		{
			name: "exposed-error",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return errors.New("handler failed")
			},
			errHandler: DefaultErrHandler(true, slog.Default()),
			wantCode:   http.StatusInternalServerError,
			wantResp:   `{"error":"handler failed","message":"Internal Server Error"}`,
		},
		{
			name: "json-marshaler-error",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return CustomError{err: "myerror"}
			},
			errHandler: DefaultErrHandler(true, slog.Default()),
			wantCode:   http.StatusInternalServerError,
			wantResp:   `{"message":"myerror"}`,
		},
		{
			name: "json-marshaler-status-coder-error",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return CustomError{
					err:  "well your're not from here are you",
					code: http.StatusUnauthorized,
				}
			},
			wantCode: http.StatusUnauthorized,
			wantResp: `{"message":"well your're not from here are you"}`,
		},
		{
			name: "default-status-coder-error",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return ErrBadRequest
			},
			wantCode: http.StatusBadRequest,
			wantResp: `{"message":"Bad Request"}`,
		},
		{
			name: "default-status-coder-with-wrap-error",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return ErrBadRequest.Wrap(errors.New("invalid payload"))
			},
			errHandler: DefaultErrHandler(true, slog.Default()),
			wantCode:   http.StatusBadRequest,
			wantResp:   `{"error":"invalid payload","message":"Bad Request"}`,
		},
		{
			name: "default-HTTPError",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return NewHTTPError(http.StatusNotFound, "user not found")
			},
			wantCode: http.StatusNotFound,
			wantResp: `{"message":"user not found"}`,
		},
		{
			name: "default-HTTPError-with-wrap",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return NewHTTPError(
					http.StatusNotFound,
					"user not found",
				).Wrap(errors.New("postgres: user with ID 1 is not present"))
			},
			errHandler: DefaultErrHandler(true, slog.Default()),
			wantCode:   http.StatusNotFound,
			wantResp:   `{"error":"postgres: user with ID 1 is not present","message":"user not found"}`,
		},
		{
			name: "method-head-error",
			handler: func(_ http.ResponseWriter, _ *http.Request) error {
				return NewHTTPError(
					http.StatusNotFound,
					"user not found",
				).Wrap(errors.New("postgres: user with ID 1 is not present"))
			},
			errHandler: DefaultErrHandler(true, slog.Default()),
			wantCode:   http.StatusNotFound,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()

			h := New()
			if tt.errHandler != nil {
				h.HTTPErrorHandler = tt.errHandler
			}

			h.Get("/", tt.handler)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.name == "method-head-error" {
				req = httptest.NewRequest(http.MethodHead, "/", nil)
			}
			h.ServeHTTP(rc, req)

			Equal(t, tt.wantResp, rc.Body.String())
			Equal(t, tt.wantCode, rc.Code)
		})
	}
}

type CustomError struct {
	err  string
	code int
}

func (e CustomError) Error() string {
	return e.err
}

func (e CustomError) MarshalJSON() ([]byte, error) {
	return fmt.Appendf([]byte{}, `{"message":"%s"}`, e.err), nil
}

func (e CustomError) StatusCode() int {
	return e.code
}
