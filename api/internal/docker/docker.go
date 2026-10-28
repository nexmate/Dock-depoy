// Package docker queries the Docker Engine for compose containers and their logs.
package docker

import (
	"bufio"
	"context"
	"io"
	"sort"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const (
	projectLabel = "com.docker.compose.project"
	serviceLabel = "com.docker.compose.service"
)

type Port struct {
	Public  uint16 `json:"public"`
	Private uint16 `json:"private"`
	Proto   string `json:"proto"`
}

type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Service string `json:"service"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Image   string `json:"image"`
	Ports   []Port `json:"ports"`
}

type Client interface {
	// ProjectContainers returns all compose containers keyed by project name.
	ProjectContainers(ctx context.Context) (map[string][]Container, error)
	// Logs follows a container's logs until ctx is done.
	Logs(ctx context.Context, id string, tail int, out func(line string)) error
}

// Status summarises a stack from its containers.
func Status(cs []Container) string {
	if len(cs) == 0 {
		return "inactive"
	}
	running := 0
	for _, c := range cs {
		if c.State == "running" {
			running++
		}
	}
	switch {
	case running == len(cs):
		return "running"
	case running > 0:
		return "partial"
	default:
		return "stopped"
	}
}

type engine struct{ cli *client.Client }

func NewClient() (Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &engine{cli: cli}, nil
}

func (e *engine) ProjectContainers(ctx context.Context) (map[string][]Container, error) {
	list, err := e.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", projectLabel)),
	})
	if err != nil {
		return nil, err
	}
	out := map[string][]Container{}
	for _, c := range list {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ports := []Port{}
		seen := map[Port]bool{}
		for _, p := range c.Ports {
			pp := Port{Public: p.PublicPort, Private: p.PrivatePort, Proto: p.Type}
			if !seen[pp] { // IPv4 and IPv6 bindings duplicate each port
				seen[pp] = true
				ports = append(ports, pp)
			}
		}
		project := c.Labels[projectLabel]
		out[project] = append(out[project], Container{
			ID: c.ID, Name: name, Service: c.Labels[serviceLabel],
			State: c.State, Status: c.Status, Image: c.Image, Ports: ports,
		})
	}
	for _, cs := range out {
		sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
	}
	return out, nil
}

func (e *engine) Logs(ctx context.Context, id string, tail int, out func(string)) error {
	info, err := e.cli.ContainerInspect(ctx, id)
	if err != nil {
		return err
	}
	rc, err := e.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true, ShowStderr: true, Follow: true, Timestamps: false,
		Tail: itoa(tail),
	})
	if err != nil {
		return err
	}
	defer rc.Close()
	var r io.Reader = rc
	if info.Config == nil || !info.Config.Tty {
		pr, pw := io.Pipe()
		go func() {
			_, err := stdcopy.StdCopy(pw, pw, rc)
			pw.CloseWithError(err)
		}()
		r = pr
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		out(strings.TrimRight(sc.Text(), "\r"))
	}
	if ctx.Err() != nil {
		return nil
	}
	return sc.Err()
}

func itoa(n int) string {
	if n <= 0 {
		return "all"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
