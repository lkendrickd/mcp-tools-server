package server

import (
	"log/slog"
	"net/http"
	"net/url"
)

// SecurityManager provides security-related checks for HTTP servers.
type SecurityManager struct {
	allowedOrigins    []string
	enableOriginCheck bool
	logger            *slog.Logger
}

// NewSecurityManager creates a new SecurityManager.
func NewSecurityManager(allowedOrigins []string, enableOriginCheck bool, logger *slog.Logger) *SecurityManager {
	return &SecurityManager{
		allowedOrigins:    allowedOrigins,
		enableOriginCheck: enableOriginCheck,
		logger:            logger,
	}
}

// OriginCheckMiddleware is a middleware that validates the Origin header.
func (sm *SecurityManager) OriginCheckMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sm.enableOriginCheck {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if origin == "" {
			sm.logger.Warn("Security check: Rejecting request with missing Origin header")
			http.Error(w, "Forbidden: Missing Origin header", http.StatusForbidden)
			return
		}

		originURL, err := url.Parse(origin)
		if err != nil {
			sm.logger.Warn("Security check: Rejecting request with invalid Origin header", "origin", origin)
			http.Error(w, "Forbidden: Invalid Origin header", http.StatusForbidden)
			return
		}

		// Normalize the origin by removing the port if it's a standard one.
		hostname := originURL.Hostname()

		isAllowed := false
		for _, allowed := range sm.allowedOrigins {
			if allowed == "*" || allowed == hostname {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			sm.logger.Warn("Security check: Rejecting request from disallowed origin", "origin", origin)
			http.Error(w, "Forbidden: Origin not allowed", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
