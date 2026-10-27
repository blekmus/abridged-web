package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 120 * time.Second
	maxHeaderBytes    = 32 << 10
	shutdownTimeout   = 10 * time.Second
)

// NewHTTPServer builds the server used in production. WriteTimeout is left at
// zero on purpose: Go applies it to the whole response, so any finite value
// would cut long video streams short. nginx/BunkerWeb own slow-client
// enforcement; the other timeouts still protect the listener itself.
func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      0,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

// ShutdownServer drains active requests for shutdownTimeout, then force-closes
// anything still running (long video streams are the expected case).
func ShutdownServer(srv *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		if closeErr := srv.Close(); closeErr != nil {
			return fmt.Errorf("shutdown: %w (force close: %v)", err, closeErr)
		}
		return err
	}

	return nil
}
