package server

import "github.com/gin-gonic/gin"

// The analytics script is loaded from this origin, and it posts its payloads to
// the same origin. Keep both in sync if the analytics host changes.
const analyticsOrigin = "https://analytics.dinil.dev"

const contentSecurityPolicy = "default-src 'self'; " +
	"base-uri 'self'; " +
	"object-src 'none'; " +
	"frame-ancestors 'none'; " +
	"frame-src 'none'; " +
	"form-action 'self'; " +
	"script-src 'self' " + analyticsOrigin + "; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; " +
	"font-src 'self'; " +
	"media-src 'self' blob:; " +
	"connect-src 'self' " + analyticsOrigin + "; " +
	"worker-src 'self' blob:; " +
	"manifest-src 'self'"

const permissionsPolicy = "accelerometer=(), " +
	"autoplay=(self), " +
	"browsing-topics=(), " +
	"camera=(), " +
	"display-capture=(), " +
	"encrypted-media=(self), " +
	"fullscreen=(self), " +
	"geolocation=(), " +
	"gyroscope=(), " +
	"magnetometer=(), " +
	"microphone=(), " +
	"midi=(), " +
	"payment=(), " +
	"picture-in-picture=(self), " +
	"publickey-credentials-get=(), " +
	"screen-wake-lock=(self), " +
	"serial=(), " +
	"usb=(), " +
	"web-share=(self), " +
	"xr-spatial-tracking=()"

// securityHeaders is the app-layer baseline. HSTS is intentionally left to the
// TLS terminator (BunkerWeb/nginx); setting it here as well creates duplicate
// headers with potentially different max-age/includeSubDomains values.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("Content-Security-Policy", contentSecurityPolicy)
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		headers.Set("Permissions-Policy", permissionsPolicy)
		headers.Set("Cross-Origin-Opener-Policy", "same-origin")
		headers.Set("Cross-Origin-Resource-Policy", "same-origin")
		c.Next()
	}
}
