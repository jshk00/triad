package triad

import (
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"
)

const defaultReadTimeout = 30 * time.Second

type Server struct {
	srv            *http.Server
	OnShutdown     func()
	Address        string
	ErrorLog       *log.Logger
	Logger         *slog.Logger
	Listener       net.Listener
	TLSConfig      *tls.Config
	ListnerNetwork string
	ReadTimeout    time.Duration
	HideBanner     bool
	HideRoutePrint bool
	HidePort       bool
}

func (s *Server) StartTLS(
	ctx context.Context,
	h http.Handler,
	certPath, keyPath string,
) error {
	cert, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}
	if s.TLSConfig == nil {
		s.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2", "http/1.1"},
		}
	}
	s.TLSConfig.Certificates = []tls.Certificate{certificate}
	return s.Start(ctx, h)
}

func (s *Server) Start(ctx context.Context, h http.Handler) error {
	if s.ReadTimeout == 0 {
		s.ReadTimeout = defaultReadTimeout
	}
	if s.ErrorLog == nil {
		s.ErrorLog = slog.NewLogLogger(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}),
			slog.LevelError,
		)
	}
	if s.Logger == nil {
		s.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{}))
	}

	s.srv = &http.Server{
		Handler:     h,
		ReadTimeout: s.ReadTimeout,
		ErrorLog:    s.ErrorLog,
		Addr:        s.Address,
	}

	if s.OnShutdown != nil {
		s.srv.RegisterOnShutdown(s.OnShutdown)
	}

	if s.Listener == nil {
		s.ListnerNetwork = cmp.Or(s.ListnerNetwork, "tcp")
		ln, err := (&net.ListenConfig{}).Listen(ctx, s.ListnerNetwork, s.Address)
		if err != nil {
			return err
		}
		s.Listener = ln
		if s.TLSConfig != nil {
			s.Listener = tls.NewListener(ln, s.TLSConfig)
		}
	}

	if !s.HideBanner {
		s.Logger.Info(
			"traid router",
			slog.String("version", version),
			slog.String("website", website),
		)
	}

	if !s.HideRoutePrint {
		if h, ok := (h).(*Triad); ok {
			for r := range h.routes.Iter() {
				s.Logger.Info(
					"route",
					slog.String("pattern", r.Pattern),
					slog.String("method", r.Method),
					slog.String("handler", r.Handler),
					slog.Any("middleware", r.Middleware),
				)
			}
		}
	}

	if !s.HidePort {
		s.Logger.Info("server started successfully", slog.String("address", s.srv.Addr))
	}

	if err := s.srv.Serve(s.Listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully shutdowns the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// Close closes the server without taking care of active connections.
func (s *Server) Close() error {
	return s.srv.Close()
}
