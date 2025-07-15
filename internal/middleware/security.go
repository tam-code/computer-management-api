package middleware

import (
	"computer-management-api/internal/config"
	"context"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// SecurityMiddleware holds security-related middleware
type SecurityMiddleware struct {
	config      *config.SecurityConfig
	rateLimiter *rate.Limiter
	mu          sync.Mutex
	clients     map[string]*rate.Limiter
}

// NewSecurityMiddleware creates a new security middleware with the given config
func NewSecurityMiddleware(cfg *config.SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		config:      cfg,
		rateLimiter: rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst),
		clients:     make(map[string]*rate.Limiter),
	}
}

// RateLimit applies rate limiting per client IP
func (sm *SecurityMiddleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := sm.getClientIP(r)

		sm.mu.Lock()
		limiter, exists := sm.clients[clientIP]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(sm.config.RateLimitRPS), sm.config.RateLimitBurst)
			sm.clients[clientIP] = limiter
		}
		sm.mu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// CORS handles Cross-Origin Resource Sharing
func (sm *SecurityMiddleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sm.config.EnableCORS {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if sm.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequestTimeout applies timeout to requests
func (sm *SecurityMiddleware) RequestTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), sm.config.RequestTimeout)
		defer cancel()

		r = r.WithContext(ctx)

		// Channel to capture if handler completes
		done := make(chan bool, 1)

		go func() {
			next.ServeHTTP(w, r)
			done <- true
		}()

		select {
		case <-done:
			// Handler completed successfully
			return
		case <-ctx.Done():
			// Request timed out
			if ctx.Err() == context.DeadlineExceeded {
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
			}
			return
		}
	})
}

// TrustedProxy handles trusted proxy headers for real IP detection
func (sm *SecurityMiddleware) TrustedProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set real IP in context for logging and rate limiting
		realIP := sm.getClientIP(r)
		ctx := context.WithValue(r.Context(), "client_ip", realIP)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders adds common security headers
func (sm *SecurityMiddleware) SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the real client IP from the request
func (sm *SecurityMiddleware) getClientIP(r *http.Request) string {
	// Check if request comes from trusted proxy
	remoteAddr := r.RemoteAddr
	if colonIndex := strings.LastIndex(remoteAddr, ":"); colonIndex != -1 {
		remoteAddr = remoteAddr[:colonIndex]
	}

	// If from trusted proxy, check forwarded headers
	if sm.isTrustedProxy(remoteAddr) {
		// Try X-Forwarded-For first
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}

		// Try X-Real-IP
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return strings.TrimSpace(xri)
		}
	}

	// Return remote address
	return remoteAddr
}

// isTrustedProxy checks if the given IP is in the trusted proxies list
func (sm *SecurityMiddleware) isTrustedProxy(ip string) bool {
	for _, trustedIP := range sm.config.TrustedProxies {
		if ip == trustedIP {
			return true
		}
	}
	return false
}

// isOriginAllowed checks if the origin is in the allowed origins list
func (sm *SecurityMiddleware) isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range sm.config.AllowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}
