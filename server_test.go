package triad

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestListener(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func TestServerStartDefaults(t *testing.T) {
	ln := newTestListener(t)
	defer ln.Close()

	s := &Server{
		Listener: ln,
	}

	done := make(chan error, 1)
	go func() {
		done <- s.Start(context.Background(), http.NewServeMux())
	}()

	time.Sleep(100 * time.Millisecond)

	if s.srv == nil {
		t.Fatal("server not initialized")
	}
	if s.ReadTimeout != defaultReadTimeout {
		t.Fatalf("ReadTimeout=%v want=%v", s.ReadTimeout, defaultReadTimeout)
	}
	if s.ErrorLog == nil {
		t.Fatal("ErrorLog not initialized")
	}

	if err := s.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not exit")
	}
}

func TestServerOnShutdown(t *testing.T) {
	ln := newTestListener(t)
	defer ln.Close()

	called := make(chan struct{}, 1)

	s := &Server{
		Listener:       ln,
		HideBanner:     true,
		HidePort:       true,
		HideRoutePrint: true,
		OnShutdown: func() {
			called <- struct{}{}
		},
	}

	go s.Start(context.Background(), http.NewServeMux())

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("OnShutdown was not called")
	}
}

func TestServerInvalidAddress(t *testing.T) {
	s := &Server{
		Address:        ":-1",
		HideBanner:     true,
		HidePort:       true,
		HideRoutePrint: true,
	}

	if err := s.Start(context.Background(), http.NewServeMux()); err == nil {
		t.Fatal("expected listen error")
	}
}

func TestServerTLSListener(t *testing.T) {
	s := &Server{
		HideBanner:     true,
		HidePort:       true,
		HideRoutePrint: true,
	}

	done := make(chan error, 1)
	go func() {
		done <- s.StartTLS(context.Background(), http.NewServeMux(), "testdata/ca.crt", "testdata/ca.key")
	}()

	time.Sleep(100 * time.Millisecond)

	if err := s.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatal(err)
	}

	<-done
}

func TestServerStartTLSMissingFiles(t *testing.T) {
	s := &Server{}

	if err := s.StartTLS(
		context.Background(),
		http.NewServeMux(),
		"missing.crt",
		"missing.key",
	); err == nil {
		t.Fatal("expected error")
	}
}

func TestServerStartTLSMissingKey(t *testing.T) {
	dir := t.TempDir()

	cert := filepath.Join(dir, "cert.pem")
	if err := os.WriteFile(cert, []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{}
	if err := s.StartTLS(
		context.Background(),
		http.NewServeMux(),
		cert,
		filepath.Join(dir, "missing.key"),
	); err == nil {
		t.Fatal("expected error")
	}
}

func TestServerStartTLSInvalidCertificate(t *testing.T) {
	dir := t.TempDir()

	cert := filepath.Join(dir, "cert.pem")
	key := filepath.Join(dir, "key.pem")

	os.WriteFile(cert, []byte("invalid"), 0o644)
	os.WriteFile(key, []byte("invalid"), 0o644)

	s := &Server{}

	if err := s.StartTLS(context.Background(), http.NewServeMux(), cert, key); err == nil {
		t.Fatal("expected invalid certificate error")
	}
}

func TestServerPrinting(t *testing.T) {
	r := New()
	r.Use(func(next HandlerFunc) HandlerFunc { return next })
	r.Get("/p1", func(_ http.ResponseWriter, _ *http.Request) error { return nil })
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{}))
	s := &Server{Address: ":8080", Logger: logger}
	done := make(chan error)
	go func() {
		done <- s.Start(context.Background(), r)
	}()
	time.Sleep(400 * time.Millisecond)
	NoError(t, s.Close())
	<-done
	for row := range strings.SplitSeq(buf.String(), "\n") {
		if strings.Contains(row, "version") {
			if !strings.Contains(row, "https://jshk00.github.io/triad") {
				t.Fatal("should contain website")
			}
		}
		if strings.Contains(row, `"pattern":"/p1"`) {
			if !strings.Contains(row, "github.com/jshk00/triad.TestServerPrinting.func2") {
				t.Fatal("handler should present")
			}
		}
	}
}
