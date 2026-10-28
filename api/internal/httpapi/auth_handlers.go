package httpapi

import (
	"errors"
	"net/http"

	"github.com/nenad/docker-depoy/api/internal/auth"
)

type credentialsBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *server) authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{
		"setupDone":     s.Auth.IsSetup(),
		"authenticated": s.authenticated(r),
	})
}

func (s *server) startSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    s.Sessions.Create(),
		Path:     "/",
		MaxAge:   int(s.Sessions.TTL.Seconds()),
		HttpOnly: true,
		Secure:   s.CookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *server) setup(w http.ResponseWriter, r *http.Request) {
	var b credentialsBody
	if !decode(w, r, &b) {
		return
	}
	switch err := s.Auth.Setup(b.Username, b.Password); {
	case errors.Is(err, auth.ErrAlreadySetup):
		writeError(w, http.StatusConflict, err.Error(), "")
	case errors.Is(err, auth.ErrWeakPassword), errors.Is(err, auth.ErrBadCredentials):
		writeError(w, http.StatusBadRequest, err.Error(), "")
	case err != nil:
		writeErr(w, err)
	default:
		s.startSession(w)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if !s.Limiter.Allow() {
		writeError(w, http.StatusTooManyRequests, "too many failed logins, try again in a minute", "")
		return
	}
	var b credentialsBody
	if !decode(w, r, &b) {
		return
	}
	if err := s.Auth.Verify(b.Username, b.Password); err != nil {
		if errors.Is(err, auth.ErrBadCredentials) {
			s.Limiter.Fail()
			writeError(w, http.StatusUnauthorized, err.Error(), "")
			return
		}
		writeErr(w, err)
		return
	}
	s.startSession(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		s.Sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}
