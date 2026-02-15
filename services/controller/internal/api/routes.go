package api

import (
	"net/http"

	"github.com/esuEdu/game-infra/controller/internal/app"
)

func registerRoutes(a *app.App, mux *http.ServeMux) {
	mux.Handle("GET /healthz", wrap(a, handleHealth()))
	mux.Handle("GET /v1/status", wrap(a, handleStatus()))

	mux.Handle("POST /v1/server/start", wrap(a, handleStart()))
	mux.Handle("POST /v1/server/stop", wrap(a, handleStop()))
	mux.Handle("POST /v1/server/switch", wrap(a, handleSwitch()))
	mux.Handle("POST /v1/server/backup", wrap(a, handleBackup()))
	mux.Handle("POST /v1/server/command", wrap(a, handleCommand()))

	mux.Handle("/", wrap(a, handleNotFound()))
}
