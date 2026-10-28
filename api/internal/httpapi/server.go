// Package httpapi exposes the REST + SSE API under /api.
package httpapi

import (
	"net/http"

	"github.com/nenad/docker-depoy/api/internal/auth"
	"github.com/nenad/docker-depoy/api/internal/compose"
	"github.com/nenad/docker-depoy/api/internal/docker"
	"github.com/nenad/docker-depoy/api/internal/stacks"
)

type Deps struct {
	Stacks       *stacks.Store
	Auth         *auth.Store
	Sessions     *auth.Sessions
	Limiter      *auth.Limiter
	Jobs         *compose.Manager
	Runner       compose.Runner
	Docker       docker.Client
	CookieSecure bool
}

type server struct{ Deps }

func New(d Deps) http.Handler {
	s := &server{d}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/status", s.authStatus)
	mux.HandleFunc("POST /api/auth/setup", s.setup)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)

	mux.Handle("GET /api/stacks", s.private(s.listStacks))
	mux.Handle("POST /api/stacks", s.private(s.createStack))
	mux.Handle("GET /api/stacks/{name}", s.private(s.getStack))
	mux.Handle("PUT /api/stacks/{name}", s.private(s.saveStack))
	mux.Handle("DELETE /api/stacks/{name}", s.private(s.deleteStack))
	mux.Handle("POST /api/stacks/{name}/{action}", s.private(s.runAction))
	mux.Handle("GET /api/stacks/{name}/containers", s.private(s.stackContainers))
	mux.Handle("GET /api/jobs/{id}/stream", s.private(s.streamJob))
	mux.Handle("GET /api/containers/{id}/logs", s.private(s.containerLogs))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not found", "")
	})
	return mux
}

func (s *server) authenticated(r *http.Request) bool {
	c, err := r.Cookie(auth.CookieName)
	return err == nil && s.Sessions.Valid(c.Value)
}

func (s *server) private(h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "")
			return
		}
		h(w, r)
	})
}
