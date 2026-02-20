package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/esuEdu/game-infra/controller/internal/domain"
)

type httpError struct {
	Status  int
	Message string
}

func (e httpError) Error() string { return e.Message }

func badRequest(msg string) error { return httpError{Status: http.StatusBadRequest, Message: msg} }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeError(aLog func(msg string, args ...any), w http.ResponseWriter, err error) {
	var he httpError
	if errors.As(err, &he) {
		writeJSON(w, he.Status, map[string]any{"error": he.Message})
		return
	}

	if errors.Is(err, domain.ErrUnknownGameType) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if errors.Is(err, domain.ErrNoBackupForGame) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if errors.Is(err, domain.ErrNoActiveGame) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
		return
	}

	// generic 500
	aLog("internal error", "err", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal server error"})
}
