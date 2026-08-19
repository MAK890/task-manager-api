// Package server owns HTTP server startup and graceful shutdown.
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MAK890/task-manager-api/internal/config"
)

type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	logger          *log.Logger
}

func New(cfg config.Server, handler http.Handler, logger *log.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Address,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
		logger:          logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	serverError := make(chan error, 1)
	go func() {
		s.logger.Printf("HTTP server starting: address=%s", s.httpServer.Addr)
		serverError <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			s.logger.Println("HTTP server closed normally")
			return nil
		}
		s.logger.Printf("HTTP server stopped unexpectedly: error=%v", err)
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		s.logger.Printf("HTTP server shutdown requested: reason=%v timeout=%s", ctx.Err(), s.shutdownTimeout)
		shutdownContext, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownContext); err != nil {
			s.logger.Printf("HTTP server graceful shutdown failed: error=%v", err)
			return fmt.Errorf("shut down HTTP server: %w", err)
		}
		s.logger.Println("HTTP server stopped")
		return nil
	}
}
