package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ctxKey string

const (
	ctxRequestID ctxKey = "rid"
	ctxRealIP    ctxKey = "real_ip"
)

// request id
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := newRID()
		w.Header().Set("X-Request-Id", rid)
		ctx := context.WithValue(r.Context(), ctxRequestID, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func getRID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxRequestID).(string); ok && v != "" {
		return v
	}
	return "unknown"
}

// real ip
func realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		ctx := context.WithValue(r.Context(), ctxRealIP, ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getIP(ctx context.Context) string {
	if v, ok := ctx.Value(ctxRealIP).(string); ok && v != "" {
		return v
	}
	return "unknown"
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

// access log (LOG LAYER)
func accessLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		log.Info("http request",
			"rid", getRID(r.Context()),
			"ip", getIP(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"bytes", sw.bytes,
			"dur_ms", time.Since(start).Milliseconds(),
		)
	})
}

// panic recovery
func recoverPanic(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Error("panic recovered", "rid", getRID(r.Context()), "panic", v)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// backpressure
func limitInFlight(max int, next http.Handler) http.Handler {
	sem := make(chan struct{}, max)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next.ServeHTTP(w, r)
		default:
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
		}
	})
}

// request timeout
func withTimeout(d time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// statusWriter captures status + bytes for logging
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
	mu     sync.Mutex
}

func (w *statusWriter) WriteHeader(code int) {
	w.mu.Lock()
	w.status = code
	w.mu.Unlock()
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	w.mu.Lock()
	w.bytes += n
	w.mu.Unlock()
	return n, err
}
