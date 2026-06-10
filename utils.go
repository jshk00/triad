package triad

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrStreamingClosed   = errors.New("traid: sse streaming channel closed")
	ErrClientGone        = errors.New("triad: client is disconnected")
	ErrUnsupportedStream = errors.New(
		"triad: sse streaming is not supported by http respose writer",
	)
)

// Binds the json to given body
func Bind(r *http.Request, body any) error {
	return json.NewDecoder(r.Body).Decode(body)
}

// JSON writes json response
func JSON(w http.ResponseWriter, body any, status int) error {
	w.Header().Set(HeaderContentType, MIMEApplicationJSON)
	w.WriteHeader(status)
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	w.Write(b)
	return nil
}

// XML writes xml response
func XML(w http.ResponseWriter, body any, status int) error {
	w.Header().Set(HeaderContentType, MIMETextXML)
	w.WriteHeader(status)
	return xml.NewEncoder(w).Encode(body)
}

// Text write plain text response
func Text(w http.ResponseWriter, body string, status int) error {
	w.Header().Set(HeaderContentType, MIMETextPlain)
	w.WriteHeader(status)
	_, err := fmt.Fprint(w, body)
	return err
}

// HTML writes html response
func HTML(w http.ResponseWriter, body string, status int) error {
	w.Header().Set(HeaderContentType, MIMETextHTML)
	w.WriteHeader(status)
	_, err := fmt.Fprint(w, body)
	return err
}

// Event hold all the fields required by SSE event
type Event struct {
	ID    string
	Event string
	Data  string
	Retry int
	// Multiline determines if we should handle the multiline data field or not if true it handles
	// the Multiline data field by scanning byte loop
	Multiline bool
}

// WriteTo implements [io.WriterTo].
//
// Event Format Example -
//
// id: 1
// event: update
// retry: 50
// Data: { "user": "alex" }
func (e *Event) WriteTo(w io.Writer) (n int64, err error) {
	var nn int

	if e.ID != "" {
		nn, err = fmt.Fprintf(w, "id: %s\n", e.ID)
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}
	if e.Event != "" {
		nn, err = fmt.Fprintf(w, "event: %s\n", e.Event)
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}
	if e.Retry > 0 {
		nn, err = fmt.Fprintf(w, "retry: %d\n", e.Retry)
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}

	// Handles multiline input using simple byte scan loop
	start := 0
	if e.Multiline {
		for i := 0; i < len(e.Data); i++ {
			if e.Data[i] == '\n' {
				nn, err = fmt.Fprintf(w, "data: %s\n", e.Data[start:i])
				n += int64(nn)
				if err != nil {
					return n, err
				}
				start = i + 1
			}
		}
	}
	// Flush remaining data after last newline in byte scan loop
	nn, err = fmt.Fprintf(w, "data: %s\n", e.Data[start:])
	n += int64(nn)
	if err != nil {
		return n, err
	}

	nn, err = io.WriteString(w, "\n")
	n += int64(nn)
	return n, err
}

// SSE sends event stream over the network
func SSE(w http.ResponseWriter, r *http.Request, ch <-chan io.WriterTo) error {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return ErrUnsupportedStream
	}

	for {
		select {
		case <-r.Context().Done():
			return ErrClientGone
		case e, ok := <-ch:
			if !ok {
				return ErrStreamingClosed
			}
			if _, err := e.WriteTo(w); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

// File sends file response from commly supported file types
// [MDN](https://developer.mozilla.org/en-US/docs/Web/HTTP/MIME_types/Common_types)
// For unknow filetype octet-stream is sent
// TODO: filetype logic
func File() {}

var decoders = map[reflect.Type]func(s string) (any, error){
	reflect.TypeFor[bool](): func(s string) (a any, err error) {
		return strconv.ParseBool(s)
	},
	reflect.TypeFor[int](): func(s string) (any, error) {
		return strconv.Atoi(s)
	},
	reflect.TypeFor[int64](): func(s string) (any, error) {
		return strconv.ParseInt(s, 10, 64)
	},
	reflect.TypeFor[uint64](): func(s string) (any, error) {
		return strconv.ParseUint(s, 10, 64)
	},
	reflect.TypeFor[string](): func(s string) (any, error) {
		return s, nil
	},
	reflect.TypeFor[float64](): func(s string) (any, error) {
		return strconv.ParseFloat(s, 64)
	},
}

// RegisterDecoder adds user defined decoder for query and path params.
// WARN: RegisterDecoder is not concurrency-safe. Register all custom
// decoders during application initialization before using [PathValue]
// or [QueryValue].
func RegisterDecoder(typ reflect.Type, fn func(s string) (any, error)) {
	decoders[typ] = fn
}

// PathValue parses path param for given generic type
func PathValue[T any](r *http.Request, key string) (T, error) {
	return parse[T](r.PathValue, key)
}

// QueryValue parses query param for given generic type
func QueryValue[T any](r *http.Request, key string) (T, error) {
	return parse[T](r.URL.Query().Get, key)
}

// parse parses parameter using [reflect.Type]
func parse[T any](fn func(string) string, key string) (T, error) {
	var zero T
	if strings.TrimSpace(key) == "" {
		return zero, errors.New("invalid empty key")
	}

	s := fn(key)
	if s == "" {
		return zero, fmt.Errorf("%s key is not found", key)
	}

	typ := reflect.TypeFor[T]()
	dec, ok := decoders[typ]
	if !ok {
		return zero, fmt.Errorf("%v doesn't have registered decoder", typ)
	}
	v, err := dec(s)
	if err != nil {
		return zero, fmt.Errorf("decoder error: %w", err)
	}
	vv, ok := v.(T)
	if ok {
		return vv, nil
	}
	return zero, fmt.Errorf("returned type is not of %v", typ)
}
