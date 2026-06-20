package triad

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type sample struct {
	Name string `json:"name" xml:"name"`
}

type badJSON struct {
	Ch chan int
}

type badXML struct {
	Ch chan int `xml:"ch"`
}

func TestBind(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid", `{"name":"john doe"}`, false},
		{"invalid", `{`, true},
		{"empty", ``, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			var got sample
			err := Bind(req, &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got.Name != "john doe" {
				t.Fatalf("unexpected value %#v", got)
			}
		})
	}
}

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()
	NoError(t, JSON(w, sample{Name: "john doe"}, http.StatusCreated))
	Equal(t, http.StatusCreated, w.Code)
	Equal(t, MIMEApplicationJSON, w.Header().Get(HeaderContentType))
	var s sample
	NoError(t, json.Unmarshal(w.Body.Bytes(), &s))
	Equal(t, "john doe", s.Name)
	NotNil(t, JSON(httptest.NewRecorder(), badJSON{}, http.StatusOK))
}

func TestXML(t *testing.T) {
	w := httptest.NewRecorder()
	NoError(t, XML(w, sample{Name: "john doe"}, http.StatusOK))
	Equal(t, MIMETextXML, w.Header().Get(HeaderContentType))
	var s sample
	NoError(t, xml.Unmarshal(w.Body.Bytes(), &s))
	Equal(t, "john doe", s.Name)
	NotNil(t, XML(httptest.NewRecorder(), badXML{}, http.StatusOK))
}

func TestTextHTML(t *testing.T) {
	cases := []struct {
		name, body, ct string
		fn             func(http.ResponseWriter, string, int) error
	}{
		{"text", "hello", MIMETextPlain, Text},
		{"html", "<h1>x</h1>", MIMETextHTML, HTML},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			NoError(t, tt.fn(w, tt.body, http.StatusAccepted))
			Equal(t, http.StatusAccepted, w.Code)
			Equal(t, tt.ct, w.Header().Get(HeaderContentType))
			Equal(t, tt.body, w.Body.String())
		})
	}
}

func TestEventWriteTo(t *testing.T) {
	tests := []struct {
		name string
		ev   Event
		want string
	}{
		{"data", Event{Data: "hello"}, "data: hello\n\n"},
		{
			"all",
			Event{ID: "1", Event: "update", Retry: 5, Data: "x"},
			"id: 1\nevent: update\nretry: 5\ndata: x\n\n",
		},
		{
			"multi",
			Event{Data: "a\nb\nc", Multiline: true},
			"data: a\ndata: b\ndata: c\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			n, err := tt.ev.WriteTo(&b)
			NoError(t, err)
			Equal(t, tt.want, b.String())
			Equal(t, int64(len(tt.want)), n)
		})
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("boom") }

func TestEventWriteToError(t *testing.T) {
	_, err := (&Event{Data: "x"}).WriteTo(errWriter{})
	NotNil(t, err)
}

type fakeRW struct{ h http.Header }

func (f *fakeRW) Header() http.Header {
	if f.h == nil {
		f.h = make(http.Header)
	}
	return f.h
}
func (*fakeRW) Write([]byte) (int, error) { return 0, nil }
func (*fakeRW) WriteHeader(int)           {}

type wt struct {
	err error
}

func (w wt) WriteTo(io.Writer) (int64, error) { return 0, w.err }

func TestSSE(t *testing.T) {
	t.Run("unsupported", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ch := make(chan io.WriterTo)
		Equal(t, SSE(&fakeRW{}, req, ch), ErrUnsupportedStream)
	})

	t.Run("closed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ch := make(chan io.WriterTo)
		close(ch)
		Equal(t, SSE(httptest.NewRecorder(), req, ch), ErrStreamingClosed)
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		ch := make(chan io.WriterTo)
		Equal(t, SSE(httptest.NewRecorder(), req, ch), ErrClientGone)
	})

	t.Run("writer error", func(t *testing.T) {
		boom := errors.New("boom")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		ch := make(chan io.WriterTo, 1)
		ch <- wt{errors.New("boom")}
		Equal(t, SSE(httptest.NewRecorder(), req, ch), boom)
	})
}

func TestStream(t *testing.T) {
	w := httptest.NewRecorder()
	if err := Stream(w, strings.NewReader("hello"), 5, "text/plain"); err != nil {
		t.Fatal(err)
	}
	if w.Body.String() != "hello" {
		t.Fatal("body")
	}
	if w.Header().Get(HeaderContentLength) != "5" {
		t.Fatal("length")
	}
}

func TestFileAttachment(t *testing.T) {
	content, err := os.ReadFile("testdata/sample.csv")
	NoError(t, err)

	t.Run("file stream", func(t *testing.T) {
		w := httptest.NewRecorder()
		NoError(t, File(w, "testdata/sample.csv"))
		Equal(t, w.Body.Bytes(), content)
		Equal(t, strconv.Itoa(len(content)), w.Header().Get(HeaderContentLength))
		Equal(t, "text/csv; charset=utf-8", w.Header().Get(HeaderContentType))
	})

	t.Run("attachment stream", func(t *testing.T) {
		w := httptest.NewRecorder()
		NoError(t, Attachment(w, "testdata/sample.csv"))
		Equal(t, w.Body.Bytes(), content)
		Equal(t, `attachment; filename="sample.csv"`, w.Header().Get(HeaderContentDisposition))
		Equal(t, strconv.Itoa(len(content)), w.Header().Get(HeaderContentLength))
		Equal(t, "text/csv; charset=utf-8", w.Header().Get(HeaderContentType))
	})
}

type MyInt int

func TestPathAndQueryValue(t *testing.T) {
	RegisterDecoder(reflect.TypeFor[MyInt](), func(_ string) (any, error) {
		return MyInt(99), nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.SetPathValue("id", "42")

	v, err := PathValue[int](req, "id")
	if err != nil || v != 42 {
		t.Fatal(v, err)
	}

	cv, err := PathValue[MyInt](req, "id")
	if err != nil || cv != 99 {
		t.Fatal(cv, err)
	}

	u := &url.URL{}
	q := u.Query()
	q.Set("name", "jay")
	q.Set("num", "10")
	u.RawQuery = q.Encode()
	req.URL = u

	s, err := QueryValue[string](req, "name")
	if err != nil || s != "jay" {
		t.Fatal(s, err)
	}

	n, err := QueryValue[int](req, "num")
	if err != nil || n != 10 {
		t.Fatal(n, err)
	}

	if _, err := QueryValue[int](req, "missing"); err == nil {
		t.Fatal("expected missing key error")
	}
	if _, err := QueryValue[int](req, " "); err == nil {
		t.Fatal("expected empty key error")
	}
}

func TestFileMIMEFallback(t *testing.T) {
	content, err := os.ReadFile("testdata/sample")
	NoError(t, err)
	w := httptest.NewRecorder()
	NoError(t, File(w, "testdata/sample"))
	Equal(t, string(content), w.Body.String())
	Equal(t, http.DetectContentType(content), w.Header().Get(HeaderContentType))
	Equal(t, strconv.Itoa(len(content)), w.Header().Get(HeaderContentLength))
}
