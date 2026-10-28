package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/nenad/docker-depoy/api/internal/compose"
	"github.com/nenad/docker-depoy/api/internal/stacks"
)

type apiError struct {
	Error  string `json:"error"`
	Detail string `json:"detail,omitempty"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, code int, msg, detail string) {
	writeJSON(w, code, apiError{Error: msg, Detail: detail})
}

// writeErr maps known domain errors to status codes; anything else is a 500.
func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, stacks.ErrInvalidName):
		writeError(w, http.StatusBadRequest, err.Error(), "")
	case errors.Is(err, stacks.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error(), "")
	case errors.Is(err, stacks.ErrExists), errors.Is(err, compose.ErrBusy):
		writeError(w, http.StatusConflict, err.Error(), "")
	case errors.Is(err, compose.ErrUnknownAction):
		writeError(w, http.StatusBadRequest, err.Error(), "")
	default:
		log.Printf("internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error", err.Error())
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body", err.Error())
		return false
	}
	return true
}
