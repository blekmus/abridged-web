package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"abridged-web/backend/catalog"
	"abridged-web/backend/server"
)

func main() {
	logger, err := newLogger()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	archiveRoot := envOr("ARCHIVE_ROOT", "../archive")
	port := envOr("PORT", "8080")
	appEnv := strings.ToLower(envOr("APP_ENV", "development"))
	serveFromDisk := appEnv == "development"
	publicBaseURL := envOr("PUBLIC_BASE_URL", "")

	slog.Info(
		"starting abridged-web",
		"archive_root", archiveRoot,
		"env", appEnv,
		"port", port,
		"public_base_url", publicBaseURL,
	)

	library, err := catalog.Load(archiveRoot)
	if err != nil {
		return err
	}

	router, err := server.NewRouter(library, server.Options{
		AppEnv:        appEnv,
		ServeFromDisk: serveFromDisk,
		PublicBaseURL: publicBaseURL,
	})
	if err != nil {
		return fmt.Errorf("create router: %w", err)
	}

	srv := server.NewHTTPServer(":"+port, router)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("abridged-web listening", "addr", ":"+port, "archive_root", archiveRoot, "env", appEnv)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		// Restore default signal handling so a second signal terminates
		// immediately instead of waiting for the drain window.
		stop()
		slog.Info("shutdown signal received")
	}

	if err := server.ShutdownServer(srv); err != nil {
		slog.Warn("graceful shutdown failed", "err", err)
	}

	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	slog.Info("shutdown complete")
	return nil
}

func envOr(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func newLogger() (*slog.Logger, error) {
	level, err := parseLogLevel(envOr("LOG_LEVEL", "info"))
	if err != nil {
		return nil, err
	}

	format := strings.ToLower(envOr("LOG_FORMAT", defaultLogFormat()))
	options := &slog.HandlerOptions{Level: level}

	switch format {
	case "json":
		return slog.New(slog.NewJSONHandler(os.Stdout, options)), nil
	case "text":
		return slog.New(slog.NewTextHandler(os.Stdout, options)), nil
	default:
		return nil, fmt.Errorf("invalid LOG_FORMAT %q (want json or text)", format)
	}
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q (want debug, info, warn, or error)", value)
	}
}

func defaultLogFormat() string {
	if strings.EqualFold(envOr("APP_ENV", "development"), "production") {
		return "json"
	}
	return "text"
}
