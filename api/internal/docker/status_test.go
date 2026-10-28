package docker

import "testing"

func TestStatus(t *testing.T) {
	run := Container{State: "running"}
	exited := Container{State: "exited"}
	cases := []struct {
		name string
		cs   []Container
		want string
	}{
		{"none", nil, "inactive"},
		{"all running", []Container{run, run}, "running"},
		{"mixed", []Container{run, exited}, "partial"},
		{"all exited", []Container{exited, exited}, "stopped"},
	}
	for _, c := range cases {
		if got := Status(c.cs); got != c.want {
			t.Errorf("%s: Status = %q, want %q", c.name, got, c.want)
		}
	}
}
