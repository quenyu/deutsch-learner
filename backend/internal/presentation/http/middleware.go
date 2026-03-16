package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type middleware func(http.Handler) http.Handler

type contextKey string

const requestIDContextKey contextKey = "request_id"

func chain(handler http.Handler, middlewares ...middleware) http.Handler {
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}
	return wrapped
}

func withRecovery() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					requestID := requestIDFromContext(r.Context())
					log.Printf("panic recovered request_id=%s err=%v", requestID, recovered)
					writeJSON(w, http.StatusInternalServerError, apiError{Message: "internal server error"})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func withRequestID() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set("X-Request-ID", requestID)
			ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func withLogging(slowThreshold time.Duration) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(recorder, r)

			duration := time.Since(start)
			requestID := requestIDFromContext(r.Context())
			clientIP := extractClientIP(r)
			slow := slowThreshold > 0 && duration >= slowThreshold
			log.Printf(
				"http request_id=%s method=%s path=%s status=%d bytes=%d duration_ms=%d ip=%s slow=%t",
				requestID,
				r.Method,
				r.URL.Path,
				recorder.status,
				recorder.bytes,
				duration.Milliseconds(),
				clientIP,
				slow,
			)
		})
	}
}

func withSecurityHeaders() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			if isHTTPSRequest(r) {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func withBodyLimit(maxBodyBytes int64) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func withHandlerTimeout(timeout time.Duration) middleware {
	if timeout <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	return func(next http.Handler) http.Handler {
		timeoutHandler := http.TimeoutHandler(next, timeout, `{"message":"request timed out"}`)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			timeoutHandler.ServeHTTP(w, r)
		})
	}
}

func withConcurrencyLimit(maxConcurrent int) middleware {
	if maxConcurrent <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	semaphore := make(chan struct{}, maxConcurrent)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
				next.ServeHTTP(w, r)
			default:
				w.Header().Set("Retry-After", "1")
				writeJSON(w, http.StatusServiceUnavailable, apiError{Message: "server is busy, please retry"})
			}
		})
	}
}

func withRateLimit(limiter *fixedWindowLimiter) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := extractClientIP(r)
			if !limiter.Allow(clientIP) {
				w.Header().Set("Retry-After", strconvInt(int(limiter.Window().Seconds())))
				writeJSON(w, http.StatusTooManyRequests, apiError{Message: "rate limit exceeded"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func withCORS(allowedOrigins []string) middleware {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				w.Header().Add("Vary", "Origin")
				if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-ID, X-Request-ID")
					w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
				} else if r.Method == http.MethodOptions {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(payload []byte) (int, error) {
	written, err := r.ResponseWriter.Write(payload)
	r.bytes += written
	return written, err
}

func extractClientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if first != "" {
			return first
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func requestIDFromContext(ctx context.Context) string {
	value, ok := ctx.Value(requestIDContextKey).(string)
	if !ok || value == "" {
		return "unknown"
	}
	return value
}

func generateRequestID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(buffer)
}

func isHTTPSRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func strconvInt(value int) string {
	return strconv.FormatInt(int64(value), 10)
}
