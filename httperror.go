package triad

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

var (
	ErrBadRequest                  = &HTTPError{Code: http.StatusBadRequest}            // 400
	ErrUnauthorized                = &HTTPError{Code: http.StatusUnauthorized}          // 401
	ErrForbidden                   = &HTTPError{Code: http.StatusForbidden}             // 403
	ErrNotFound                    = &HTTPError{Code: http.StatusNotFound}              // 404
	ErrMethodNotAllowed            = &HTTPError{Code: http.StatusMethodNotAllowed}      // 405
	ErrRequestTimeout              = &HTTPError{Code: http.StatusRequestTimeout}        // 408
	ErrStatusRequestEntityTooLarge = &HTTPError{Code: http.StatusRequestEntityTooLarge} // 413
	ErrUnsupportedMediaType        = &HTTPError{Code: http.StatusUnsupportedMediaType}  // 415
	ErrTooManyRequests             = &HTTPError{Code: http.StatusTooManyRequests}       // 429
	ErrInternalServerError         = &HTTPError{Code: http.StatusInternalServerError}   // 500
	ErrBadGateway                  = &HTTPError{Code: http.StatusBadGateway}            // 502
	ErrServiceUnavailable          = &HTTPError{Code: http.StatusServiceUnavailable}    // 503
)

type StatusCoder interface {
	StatusCode() int
}

// HTTPError represents Default error format for response
type HTTPError struct {
	Message string `json:"message"`
	Code    int    `json:"-"`
	err     error
}

// NewHTTPError returns instance of HTTPError.
// If multiple msg provided only first will be used.
func NewHTTPError(code int, msg string) *HTTPError {
	return &HTTPError{Code: code, Message: msg}
}

func (h *HTTPError) StatusCode() int {
	return h.Code
}

// Wrap sets internal error if required
func (h *HTTPError) Wrap(err error) *HTTPError {
	return &HTTPError{
		Message: h.Message,
		err:     err,
		Code:    h.Code,
	}
}

// Error implements error interface
func (h *HTTPError) Error() string {
	msg := h.Message
	if msg == "" {
		msg = http.StatusText(h.Code)
	}
	if h.err != nil {
		return fmt.Sprintf(
			"message: %s, status_code: %d, error: %s",
			msg,
			h.Code,
			h.err.Error(),
		)
	}
	return fmt.Sprintf("message: %s, status_code: %d", msg, h.Code)
}

func (h *HTTPError) Unwrap() error {
	return h.err
}

// DefaultErrHandler is a error handler func
// to write errors to [http.ResponseWriter]. by Default Serialize error to json
func DefaultErrHandler(showError bool, logger *slog.Logger) HTTPErrorHandler {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		code := http.StatusInternalServerError
		var sc StatusCoder
		if errors.As(err, &sc) {
			if c := sc.StatusCode(); c != 0 {
				code = c
			}
		}
		var result any
		switch m := sc.(type) {
		case json.Marshaler:
			result = m
		case *HTTPError:
			text := m.Message
			if text == "" {
				text = http.StatusText(code)
			}
			msg := map[string]string{"message": text}
			if showError {
				if e := m.Unwrap(); e != nil {
					msg["error"] = e.Error()
				}
			}
			result = msg
		default:
			var jm json.Marshaler
			if errors.As(err, &jm) {
				result = jm
			} else {
				msg := map[string]any{"message": http.StatusText(code)}
				if showError {
					msg["error"] = err.Error()
				}
				result = msg
			}
		}
		if r.Method == http.MethodHead {
			w.WriteHeader(code)
			return
		}
		if err := JSON(w, result, code); err != nil {
			logger.Error(
				"triad: default error handler failed to send error to client",
				slog.String("error", err.Error()),
			)
		}
	}
}
