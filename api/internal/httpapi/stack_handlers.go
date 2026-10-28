package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/nenad/docker-depoy/api/internal/compose"
	"github.com/nenad/docker-depoy/api/internal/docker"
	"github.com/nenad/docker-depoy/api/internal/stacks"
)

type containerCounts struct {
	Running int `json:"running"`
	Total   int `json:"total"`
}

type stackSummary struct {
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Containers containerCounts `json:"containers"`
}

type stackDetail struct {
	Name    string `json:"name"`
	Compose string `json:"compose"`
	Env     string `json:"env"`
	Status  string `json:"status"`
}

type stackBody struct {
	Name    string `json:"name"`
	Compose string `json:"compose"`
	Env     string `json:"env"`
}

func (s *server) projects(w http.ResponseWriter, r *http.Request) (map[string][]docker.Container, bool) {
	m, err := s.Docker.ProjectContainers(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "docker unavailable", err.Error())
		return nil, false
	}
	return m, true
}

func (s *server) listStacks(w http.ResponseWriter, r *http.Request) {
	names, err := s.Stacks.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	projects, ok := s.projects(w, r)
	if !ok {
		return
	}
	out := make([]stackSummary, 0, len(names))
	for _, n := range names {
		cs := projects[n]
		running := 0
		for _, c := range cs {
			if c.State == "running" {
				running++
			}
		}
		out = append(out, stackSummary{Name: n, Status: docker.Status(cs), Containers: containerCounts{running, len(cs)}})
	}
	writeJSON(w, http.StatusOK, out)
}

// validate runs `compose config` and writes a 422 on failure.
func (s *server) validate(w http.ResponseWriter, r *http.Request, name string, b stackBody) bool {
	if err := compose.Validate(r.Context(), s.Runner, s.Stacks.Dir(name), b.Compose, b.Env); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid compose file", err.Error())
		return false
	}
	return true
}

func (s *server) createStack(w http.ResponseWriter, r *http.Request) {
	var b stackBody
	if !decode(w, r, &b) {
		return
	}
	if !stacks.ValidName(b.Name) {
		writeErr(w, stacks.ErrInvalidName)
		return
	}
	if _, err := s.Stacks.Get(b.Name); err == nil {
		writeErr(w, stacks.ErrExists)
		return
	}
	if !s.validate(w, r, b.Name, b) {
		return
	}
	if err := s.Stacks.Create(b.Name, b.Compose, b.Env); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, stackDetail{Name: b.Name, Compose: b.Compose, Env: b.Env, Status: "inactive"})
}

func (s *server) getStack(w http.ResponseWriter, r *http.Request) {
	st, err := s.Stacks.Get(r.PathValue("name"))
	if err != nil {
		writeErr(w, err)
		return
	}
	projects, ok := s.projects(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, stackDetail{Name: st.Name, Compose: st.Compose, Env: st.Env, Status: docker.Status(projects[st.Name])})
}

func (s *server) saveStack(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.Stacks.Get(name); err != nil {
		writeErr(w, err)
		return
	}
	var b stackBody
	if !decode(w, r, &b) || !s.validate(w, r, name, b) {
		return
	}
	if err := s.Stacks.Save(name, b.Compose, b.Env); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *server) deleteStack(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.Stacks.Get(name); err != nil {
		writeErr(w, err)
		return
	}
	// A never-deployed stack makes `down` fail; that's fine, only busy blocks deletion.
	if err := s.Jobs.RunSync(r.Context(), name, s.Stacks.Dir(name), "down"); errors.Is(err, compose.ErrBusy) {
		writeErr(w, err)
		return
	}
	if err := s.Stacks.Delete(name); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) runAction(w http.ResponseWriter, r *http.Request) {
	name, action := r.PathValue("name"), r.PathValue("action")
	if _, ok := compose.ActionArgs(action); !ok {
		writeErr(w, compose.ErrUnknownAction)
		return
	}
	if _, err := s.Stacks.Get(name); err != nil {
		writeErr(w, err)
		return
	}
	job, err := s.Jobs.Start(name, s.Stacks.Dir(name), action)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"jobId": job.ID})
}

func (s *server) streamJob(w http.ResponseWriter, r *http.Request) {
	job, ok := s.Jobs.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "job not found", "")
		return
	}
	sse := newSSE(w)
	for ev := range job.Subscribe(r.Context()) {
		if ev.Done {
			sse.send("done", map[string]int{"exitCode": ev.ExitCode})
			return
		}
		sse.send("line", ev.Line)
	}
}

func (s *server) stackContainers(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, err := s.Stacks.Get(name); err != nil {
		writeErr(w, err)
		return
	}
	projects, ok := s.projects(w, r)
	if !ok {
		return
	}
	cs := projects[name]
	if cs == nil {
		cs = []docker.Container{}
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *server) containerLogs(w http.ResponseWriter, r *http.Request) {
	tail, err := strconv.Atoi(r.URL.Query().Get("tail"))
	if err != nil || tail <= 0 {
		tail = 200
	}
	sse := newSSE(w)
	err = s.Docker.Logs(r.Context(), r.PathValue("id"), tail, func(line string) { sse.send("line", line) })
	if err != nil {
		sse.send("error", err.Error())
	}
}
