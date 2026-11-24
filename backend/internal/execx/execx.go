// Package execx runs external programs (docker, kind, kubectl, helm) without
// going through a shell, so behaviour is identical on Windows and Linux.
package execx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Cmd describes one program invocation.
type Cmd struct {
	Name   string
	Args   []string
	Env    []string  // extra KEY=VALUE entries appended to the process environment
	Stdin  string    // optional stdin content
	Stream io.Writer // if set, stdout and stderr are also copied here live
}

// Runner executes commands. Tests substitute a fake.
type Runner interface {
	Run(ctx context.Context, c Cmd) (stdout string, err error)
}

// RunnerFunc adapts a function to Runner.
type RunnerFunc func(ctx context.Context, c Cmd) (string, error)

// Run implements Runner.
func (f RunnerFunc) Run(ctx context.Context, c Cmd) (string, error) { return f(ctx, c) }

// Local runs commands on the local machine.
type Local struct{}

// Run implements Runner. On failure the error carries the trimmed stderr.
func (Local) Run(ctx context.Context, c Cmd) (string, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	if c.Stdin != "" {
		cmd.Stdin = strings.NewReader(c.Stdin)
	}
	var stdout, stderr bytes.Buffer
	if c.Stream != nil {
		cmd.Stdout = io.MultiWriter(&stdout, c.Stream)
		cmd.Stderr = io.MultiWriter(&stderr, c.Stream)
	} else {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	}
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		return out, &Error{Cmd: c, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	}
	return out, nil
}

// Error is returned when a command fails.
type Error struct {
	Cmd    Cmd
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("%s %s: %v", e.Cmd.Name, strings.Join(e.Cmd.Args, " "), e.Err)
	if e.Stderr != "" {
		msg += ": " + e.Stderr
	}
	return msg
}

func (e *Error) Unwrap() error { return e.Err }

// Stderr returns the captured stderr of a failed command, or "".
func Stderr(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Stderr
	}
	return ""
}

// Output is shorthand for running a command and returning its stdout.
func Output(ctx context.Context, r Runner, name string, args ...string) (string, error) {
	return r.Run(ctx, Cmd{Name: name, Args: args})
}
