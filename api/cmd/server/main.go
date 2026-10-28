package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/nenad/docker-depoy/api/internal/auth"
	"github.com/nenad/docker-depoy/api/internal/compose"
	"github.com/nenad/docker-depoy/api/internal/docker"
	"github.com/nenad/docker-depoy/api/internal/httpapi"
	"github.com/nenad/docker-depoy/api/internal/stacks"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	addr := env("LISTEN_ADDR", ":8080")
	stacksDir := env("STACKS_DIR", "/opt/stacks")
	dataDir := env("DATA_DIR", "/data")
	for _, d := range []string{stacksDir, dataDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Fatalf("create %s: %v", d, err)
		}
	}

	dc, err := docker.NewClient()
	if err != nil {
		log.Fatalf("docker client: %v", err)
	}
	runner := compose.CLIRunner{}
	handler := httpapi.New(httpapi.Deps{
		Stacks:       &stacks.Store{Root: stacksDir},
		Auth:         &auth.Store{Path: filepath.Join(dataDir, "auth.json")},
		Sessions:     auth.NewSessions(7 * 24 * time.Hour),
		Limiter:      auth.NewLimiter(5, time.Minute, nil),
		Jobs:         compose.NewManager(runner, 10*time.Minute),
		Runner:       runner,
		Docker:       dc,
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
	})

	// No WriteTimeout: SSE streams stay open.
	srv := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Printf("docker-depoy api listening on %s (stacks: %s)", addr, stacksDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
