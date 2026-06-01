package triad

import (
	"fmt"
	"net/http"
)

// HTTPError represents Default error format for response
type HTTPError struct {
	internal   error
	Msg        string `json:"msg"`
	StatusCode int    `json:"status_code"`
	hook       func(w http.ResponseWriter)
}

// NewHTTPError returns instance of HTTPError.
// If multiple msg provided only first will be used.
func NewHTTPError(statusCode int, msg ...string) *HTTPError {
	h := &HTTPError{StatusCode: statusCode}
	if len(msg) > 0 {
		h.Msg = msg[0]
	}
	return h
}

// SetInternal sets internal error if required
func (h *HTTPError) SetInternal(err error) *HTTPError {
	h.internal = err
	return h
}

// Error implements error interface
func (h *HTTPError) Error() string {
	if h.internal != nil {
		return fmt.Sprintf(
			"msg: %s, status_code: %d, internal_error: %s",
			h.Msg,
			h.StatusCode,
			h.internal.Error(),
		)
	}
	return fmt.Sprintf("msg: %s, status_code: %d", h.Msg, h.StatusCode)
}

func (h *HTTPError) Unwrap() error {
	return h.internal
}

// WithText sets the error hook with mime plain text.
func (h *HTTPError) WithText() *HTTPError {
	h.hook = func(w http.ResponseWriter) {
		w.Header().Del(HeaderContentType)
		w.Header().Set(HeaderXContentTypeOptions, "nosniff")
		w.Header().Set(HeaderContentType, MIMETextPlain)
		w.WriteHeader(h.StatusCode)
		fmt.Fprint(w, h.Msg)
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

// defaultErrHandler is a error handler func
// to write errors to [http.ResponseWriter]
func defaultErrHandler(w http.ResponseWriter, err error) {
	if e, ok := err.(*HTTPError); ok {
		if e.hook == nil {
			e.WithJSON()
		}
		e.hook(w)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
