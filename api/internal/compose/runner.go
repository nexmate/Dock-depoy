// Package compose runs `docker compose` commands for stacks and tracks them as jobs.
package compose

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes `docker compose <args>` in dir, calling out for each output line.
// A non-zero exit is reported via exitCode; err is for failures to run at all.
type Runner interface {
	Run(ctx context.Context, dir string, out func(line string), args ...string) (exitCode int, err error)
}

// CLIRunner shells out to the docker CLI with the compose plugin.
type CLIRunner struct{}

func (CLIRunner) Run(ctx context.Context, dir string, out func(string), args ...string) (int, error) {
	full := append([]string{"compose", "--ansi", "never", "--progress", "plain"}, args...)
	cmd := exec.CommandContext(ctx, "docker", full...)
	cmd.Dir = dir
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			out(sc.Text())
		}
		_, _ = io.Copy(io.Discard, pr)
	}()
	err := cmd.Wait()
	pw.Close()
	<-scanDone
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return -1, err
	}
	return 0, nil
}

var actions = map[string][][]string{
	"up":      {{"up", "-d", "--remove-orphans"}},
	"stop":    {{"stop"}},
	"restart": {{"restart"}},
	"pull":    {{"pull"}, {"up", "-d", "--remove-orphans"}},
	"down":    {{"down"}},
}

// ActionArgs returns the compose commands to run, in order, for a stack action.
func ActionArgs(action string) ([][]string, bool) {
	a, ok := actions[action]
	return a, ok
}

// Validate checks compose (and env) with `docker compose config -q` without touching
// the stack's real files. It runs in dir when it exists so relative paths resolve,
// otherwise in a throwaway directory.
func Validate(ctx context.Context, r Runner, dir, compose, env string) error {
	work := dir
	if _, err := os.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		tmp, err := os.MkdirTemp("", "dd-validate-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		work = tmp
	}
	const file, envFile = ".dd-validate.yaml", ".dd-validate.env"
	if err := os.WriteFile(filepath.Join(work, file), []byte(compose), 0o600); err != nil {
		return err
	}
	defer os.Remove(filepath.Join(work, file))
	args := []string{"-f", file}
	if env != "" {
		if err := os.WriteFile(filepath.Join(work, envFile), []byte(env), 0o600); err != nil {
			return err
		}
		defer os.Remove(filepath.Join(work, envFile))
		args = append(args, "--env-file", envFile)
	}
	args = append(args, "config", "-q")
	var buf strings.Builder
	code, err := r.Run(ctx, work, func(l string) { buf.WriteString(l + "\n") }, args...)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("%s", strings.TrimSpace(buf.String()))
	}
	return nil
}
