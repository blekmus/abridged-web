package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	library := loadTestLibrary(t)
	router, err := NewRouter(library, Options{AppEnv: "test"})
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"/", "/api/catalog", "/missing"} {
		t.Run(target, func(t *testing.T) {
			response := serveRouter(t, router, target, nil)
			if response.Code >= http.StatusInternalServerError {
				t.Fatalf("status = %d", response.Code)
			}

			csp := response.Header().Get("Content-Security-Policy")
			for _, want := range []string{
				"default-src 'self'",
				"base-uri 'self'",
				"object-src 'none'",
				"frame-ancestors 'none'",
				"frame-src 'none'",
				"form-action 'self'",
				"script-src 'self' " + analyticsOrigin,
				"style-src 'self' 'unsafe-inline'",
				"img-src 'self' data: blob:",
				"font-src 'self'",
				"media-src 'self' blob:",
				"connect-src 'self' " + analyticsOrigin,
				"worker-src 'self' blob:",
				"manifest-src 'self'",
			} {
				if !strings.Contains(csp, want) {
					t.Fatalf("CSP missing %q: %s", want, csp)
				}
			}
			if strings.Contains(csp, "unsafe-eval") {
				t.Fatalf("CSP contains unsafe-eval: %s", csp)
			}
			if strings.Contains(csp, " script-src *") || strings.Contains(csp, "default-src *") {
				t.Fatalf("CSP contains a wildcard source: %s", csp)
			}

			permissions := response.Header().Get("Permissions-Policy")
			for _, want := range []string{"autoplay=(self)", "fullscreen=(self)", "camera=()", "microphone=()", "geolocation=()"} {
				if !strings.Contains(permissions, want) {
					t.Fatalf("Permissions-Policy missing %q: %s", want, permissions)
				}
			}

			checks := map[string]string{
				"X-Content-Type-Options":       "nosniff",
				"X-Frame-Options":              "DENY",
				"Referrer-Policy":              "strict-origin-when-cross-origin",
				"Cross-Origin-Opener-Policy":   "same-origin",
				"Cross-Origin-Resource-Policy": "same-origin",
			}
			for header, want := range checks {
				if got := response.Header().Get(header); got != want {
					t.Fatalf("%s = %q, want %q", header, got, want)
				}
			}
		})
	}
}
