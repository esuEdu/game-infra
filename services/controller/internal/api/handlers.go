package api

import (
	"net/http"
	"strings"

	"github.com/esuEdu/game-infra/controller/internal/app"
)

type appHandler func(*app.App, http.ResponseWriter, *http.Request) error

func wrap(a *app.App, h appHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		if err := h(a, w, r); err != nil {
			// include rid in logs via middleware logger fields
			writeError(a.Log.Error, w, err)
		}
	})
}

func handleHealth() appHandler {
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return nil
	}
}

func handleStatus() appHandler {
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		st, err := a.Controller.Status(r.Context())
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, st)
		return nil
	}
}

func handleStart() appHandler {
	type req struct {
		Game string `json:"game"`
	}
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		var body req
		if err := decodeJSON(w, r, &body); err != nil {
			return badRequest("invalid json body")
		}
		if body.Game == "" {
			return badRequest("missing field: game")
		}
		if err := a.Controller.Start(r.Context(), body.Game); err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"started": body.Game})
		return nil
	}
}

func handleStop() appHandler {
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		if err := a.Controller.Stop(r.Context()); err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"stopped": true})
		return nil
	}
}

func handleSwitch() appHandler {
	type req struct {
		Game string `json:"game"`
	}
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		var body req
		if err := decodeJSON(w, r, &body); err != nil {
			return badRequest("invalid json body")
		}
		if body.Game == "" {
			return badRequest("missing field: game")
		}
		if err := a.Controller.Switch(r.Context(), body.Game); err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"switched_to": body.Game})
		return nil
	}
}

func handleBackup() appHandler {
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		key, err := a.Controller.Backup(r.Context())
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"backup": key})
		return nil
	}
}

func handleCommand() appHandler {
	type req struct {
		Command string `json:"command"`
	}
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		var body req
		if err := decodeJSON(w, r, &body); err != nil {
			return badRequest("invalid json body")
		}
		if strings.TrimSpace(body.Command) == "" {
			return badRequest("missing field: command")
		}
		if err := a.Controller.Command(r.Context(), body.Command); err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, map[string]any{"sent": true})
		return nil
	}
}

func handleNotFound() appHandler {
	return func(a *app.App, w http.ResponseWriter, r *http.Request) error {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return nil
	}
}
