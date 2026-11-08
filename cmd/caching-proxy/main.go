package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fc1g/caching-proxy/internal/cache"
	"github.com/fc1g/caching-proxy/internal/config"
	"github.com/fc1g/caching-proxy/internal/proxy"
)

const (
	readTimeout    = 10 * time.Second
	writeTimeout   = 10 * time.Second
	maxHeaderBytes = 1 << 20

	shutdownTimeout = 10 * time.Second

	clientTimeout = 8 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		log.Printf("Application failed: %v", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cacheStore, err := cache.NewStore()
	if err != nil {
		return err
	}

	if cfg.ClearCache {
		if err := cacheStore.Clear(); err != nil {
			return fmt.Errorf("clear cache: %w", err)
		}
		log.Println("cache cleared")
		return nil
	}

	client := &http.Client{
		Timeout: clientTimeout,
	}
	handler := proxy.NewHandler(cfg.Origin, cacheStore, client)

	errCh := make(chan error, 1)
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Port),
		Handler:        handler,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}

	go func() {
		log.Printf("starting server on port :%d", cfg.Port)
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	log.Print("server stopped")

	return nil
}
