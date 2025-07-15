package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware provides request logging with security context
type LoggingMiddleware struct {
	logger *log.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *log.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// LogRequests logs incoming requests with security information
func (lm *LoggingMiddleware) LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get client IP from context (set by TrustedProxy middleware)
		clientIP, _ := r.Context().Value("client_ip").(string)
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		lm.logger.Printf("[%s] %s %s %d %v - IP: %s, User-Agent: %s",
			r.Method,
			r.RequestURI,
			r.Proto,
			wrapped.statusCode,
			duration,
			clientIP,
			r.UserAgent(),
		)

		// Log security events
		if wrapped.statusCode == http.StatusTooManyRequests {
			lm.logger.Printf("SECURITY: Rate limit exceeded for IP: %s", clientIP)
		}
		if wrapped.statusCode == http.StatusRequestTimeout {
			lm.logger.Printf("SECURITY: Request timeout for IP: %s", clientIP)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
