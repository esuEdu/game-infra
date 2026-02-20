package api

import (
	"net/http"
	"time"

	"github.com/esuEdu/game-infra/controller/internal/app"
)

func NewServer(a *app.App) *http.Server {
	mux := http.NewServeMux()
	registerRoutes(a, mux)

	var h http.Handler = mux

	// LOG LAYER + safety middleware (order matters)
	h = requestID(h)
	h = realIP(h)
	h = recoverPanic(a.Log, h)
	h = accessLog(a.Log, h)
	h = limitInFlight(256, h)
	h = withTimeout(10*time.Minute, h)

	return &http.Server{
		Addr:              a.Config.HTTPAddr,
		Handler:           h,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      11 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}
}
