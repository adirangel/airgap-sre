package toolchain

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var forwardingRe = regexp.MustCompile(`Forwarding from 127\.0\.0\.1:(\d+)`)

// PortForward starts `kubectl port-forward` to target (for example
// "svc/kagent-controller") on a free local port and returns the local
// address ("127.0.0.1:NNNNN"). Call stop to end it.
func (t *Toolchain) PortForward(ctx context.Context, namespace, target string, remotePort int) (addr string, stop func(), err error) {
	bin, err := t.Bin(ctx, "kubectl")
	if err != nil {
		return "", nil, err
	}
	pfCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(pfCtx, bin,
		"--kubeconfig", t.Paths.Kubeconfig(),
		"port-forward", "-n", namespace, "--address", "127.0.0.1", target, fmt.Sprintf(":%d", remotePort))
	cmd.Env = append(os.Environ(), t.Env()...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return "", nil, err
	}
	var stderr strings.Builder
	var mu sync.Mutex
	cmd.Stderr = writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		defer mu.Unlock()
		return stderr.Write(p)
	})
	if err := cmd.Start(); err != nil {
		cancel()
		return "", nil, err
	}
	done := make(chan struct{})
	stop = func() {
		cancel()
		<-done
	}

	found := make(chan string, 1)
	go func() {
		defer close(done)
		sc := bufio.NewScanner(stdout)
		sent := false
		for sc.Scan() {
			if m := forwardingRe.FindStringSubmatch(sc.Text()); m != nil && !sent {
				found <- "127.0.0.1:" + m[1]
				sent = true
			}
		}
		_ = cmd.Wait()
		if !sent {
			close(found)
		}
	}()

	select {
	case a, ok := <-found:
		if !ok {
			cancel()
			mu.Lock()
			msg := strings.TrimSpace(stderr.String())
			mu.Unlock()
			return "", nil, fmt.Errorf("kubectl port-forward %s/%s exited before it was ready: %s", namespace, target, msg)
		}
		return a, stop, nil
	case <-time.After(30 * time.Second):
		stop()
		return "", nil, fmt.Errorf("kubectl port-forward %s/%s did not become ready in 30s", namespace, target)
	case <-ctx.Done():
		stop()
		return "", nil, ctx.Err()
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }
