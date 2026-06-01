package triad

import (
	"fmt"
	"log/slog"
	"net/http"
)

var (
	ErrBadRequest                  = &HTTPError{StatusCode: http.StatusBadRequest}            // 400
	ErrUnauthorized                = &HTTPError{StatusCode: http.StatusUnauthorized}          // 401
	ErrForbidden                   = &HTTPError{StatusCode: http.StatusForbidden}             // 403
	ErrNotFound                    = &HTTPError{StatusCode: http.StatusNotFound}              // 404
	ErrMethodNotAllowed            = &HTTPError{StatusCode: http.StatusMethodNotAllowed}      // 405
	ErrRequestTimeout              = &HTTPError{StatusCode: http.StatusRequestTimeout}        // 408
	ErrStatusRequestEntityTooLarge = &HTTPError{StatusCode: http.StatusRequestEntityTooLarge} // 413
	ErrUnsupportedMediaType        = &HTTPError{StatusCode: http.StatusUnsupportedMediaType}  // 415
	ErrTooManyRequests             = &HTTPError{StatusCode: http.StatusTooManyRequests}       // 429
	ErrInternalServerError         = &HTTPError{StatusCode: http.StatusInternalServerError}   // 500
	ErrBadGateway                  = &HTTPError{StatusCode: http.StatusBadGateway}            // 502
	ErrServiceUnavailable          = &HTTPError{StatusCode: http.StatusServiceUnavailable}    // 503
)

// HTTPError represents Default error format for response
type HTTPError struct {
	err        error
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	hook       func(w http.ResponseWriter)
}

// NewHTTPError returns instance of HTTPError.
// If multiple msg provided only first will be used.
func NewHTTPError(code int, msg string) *HTTPError {
	return &HTTPError{StatusCode: code, Message: msg}
}

// Wrap sets internal error if required
func (h *HTTPError) Wrap(err error) *HTTPError {
	h.err = err
	return h
}

// Error implements error interface
func (h *HTTPError) Error() string {
	if h.Message == "" {
		h.Message = http.StatusText(h.StatusCode)
	}
	if h.err != nil {
		return fmt.Sprintf(
			"message: %s, status_code: %d, error: %s",
			h.Message,
			h.StatusCode,
			h.err.Error(),
		)
	}
	return fmt.Sprintf("message: %s, status_code: %d", h.Message, h.StatusCode)
}

func (h *HTTPError) Unwrap() error {
	return h.err
}

// WithText sets the error hook with mime plain text.
func (h *HTTPError) WithText() *HTTPError {
	h.hook = func(w http.ResponseWriter) {
		w.Header().Del(HeaderContentType)
		w.Header().Set(HeaderXContentTypeOptions, "nosniff")
		w.Header().Set(HeaderContentType, MIMETextPlain)
		w.WriteHeader(h.StatusCode)
		fmt.Fprint(w, h.Message)
	}
	return h
}

// WithJSON sets the error hook with mime json.
// This is the default hook if none is set.
// Both msg and status_code will be sent in payload.
func (h *HTTPError) WithJSON() *HTTPError {
	h.hook = func(w http.ResponseWriter) {
		if err := JSON(w, h, h.StatusCode); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	return h
}

// WithJSON sets the error hook with mime json.
func (h *HTTPError) WithXML() *HTTPError {
	h.hook = func(w http.ResponseWriter) {
		if err := XML(w, h, h.StatusCode); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	return h
}

// WithHeader sets hook with given key and val headers.
// Useful if you don't want to display any message but still
// want to send custom header and status code.
func (h *HTTPError) WithHeader(hdrs map[string]string) *HTTPError {
	h.hook = func(w http.ResponseWriter) {
		for k, v := range hdrs {
			w.Header().Set(k, v)
		}
		w.WriteHeader(h.StatusCode)
	}
	return h
}

// DefaultErrHandler is a error handler func
// to write errors to [http.ResponseWriter]. by Default Serialize error to json
func DefaultErrHandler(w http.ResponseWriter, err error) {
	if e, ok := err.(*HTTPError); ok {
		if e.hook == nil {
			e.WithJSON()
		}
		e.hook(w)
		return
	}
	if err := JSON(
		w,
		map[string]string{"msg": err.Error()},
		http.StatusInternalServerError,
	); err != nil {
		slog.Error(err.Error()) // rare case client disconnected
	}
}
