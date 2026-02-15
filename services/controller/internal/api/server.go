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
	h = withTimeout(25*time.Second, h)

	return &http.Server{
		Addr:              a.Config.HTTPAddr,
		Handler:           h,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
