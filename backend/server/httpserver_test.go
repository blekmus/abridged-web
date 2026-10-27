package server

import (
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerTimeouts(t *testing.T) {
	srv := NewHTTPServer(":8080", http.NewServeMux())

	if srv.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", srv.Addr)
	}
	if srv.ReadHeaderTimeout != readHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %v, want %v", srv.ReadHeaderTimeout, readHeaderTimeout)
	}
	if srv.ReadTimeout != readTimeout {
		t.Fatalf("ReadTimeout = %v, want %v", srv.ReadTimeout, readTimeout)
	}
	if srv.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout = %v, want 0 for streaming", srv.WriteTimeout)
	}
	if srv.IdleTimeout != idleTimeout {
		t.Fatalf("IdleTimeout = %v, want %v", srv.IdleTimeout, idleTimeout)
	}
	if srv.MaxHeaderBytes != maxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", srv.MaxHeaderBytes, maxHeaderBytes)
	}
}

func TestShutdownServerStopsServing(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := NewHTTPServer("", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(listener)
	}()

	response, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if err := ShutdownServer(srv); err != nil {
		t.Fatalf("ShutdownServer() = %v, want nil", err)
	}

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("Serve() = %v, want %v", err, http.ErrServerClosed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve() did not return after shutdown")
	}
}
