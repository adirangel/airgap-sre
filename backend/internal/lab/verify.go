package lab

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/adirangel/airgap-sre/backend/internal/kube"
)

// VerifyNamespace is where `lab verify` runs its probe pods.
const VerifyNamespace = "lab-verify"

const verifyPodTmpl = `apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  restartPolicy: Never
  containers:
    - name: probe
      image: %s
      imagePullPolicy: Always
      command: ["sh", "-c", "echo image-ran-ok"]
`

// Verify is the M1 acceptance test:
//
//  1. a pod using a mirrored image runs to completion, and
//  2. pods using real public images that were never mirrored cannot pull.
//
// imagePullPolicy is Always, so the node's image cache cannot hide a leak.
func (l *Lab) Verify(ctx context.Context) error {
	mirrored := l.Images.Group("base")
	if len(mirrored) == 0 {
		return fmt.Errorf("images.yaml has no image in group %q to verify with", "base")
	}
	okImage := mirrored[0].Ref
	blocked := l.Cfg.UnmirroredImages
	for _, b := range blocked {
		for _, i := range l.Images.Images {
			if i.Ref == b {
				return fmt.Errorf("%s is listed in images.yaml, so it cannot be used as an un-mirrored probe", b)
			}
		}
		if digest, err := UpstreamExists(ctx, b); err != nil {
			l.logf("note: could not confirm %s exists upstream from this host (%v)\n", b, err)
		} else {
			l.logf("upstream check: %s is a real public image (%s)\n", b, digest)
		}
	}

	k := l.Kube()
	// Start from a clean namespace so old pods cannot satisfy the checks.
	if err := k.DeleteNamespace(ctx, VerifyNamespace, 2*time.Minute); err != nil {
		return err
	}
	if err := k.EnsureNamespace(ctx, VerifyNamespace, nil); err != nil {
		return err
	}
	defer func() {
		// Best effort; a fresh context so cleanup still runs after Ctrl+C.
		cctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_ = k.DeleteNamespace(cctx, VerifyNamespace, 2*time.Minute)
	}()
	if err := k.Apply(ctx, fmt.Sprintf(verifyPodTmpl, "mirrored", VerifyNamespace, okImage)); err != nil {
		return err
	}
	for i, b := range blocked {
		if err := k.Apply(ctx, fmt.Sprintf(verifyPodTmpl, blockedPodName(i), VerifyNamespace, b)); err != nil {
			return err
		}
	}

	// 1. The mirrored image must run.
	err := kube.Poll(ctx, 3*time.Minute, 2*time.Second, func(ctx context.Context) (bool, string, error) {
		p, status, err := l.probePod(ctx, "mirrored")
		if err != nil || p == nil {
			return false, status, err
		}
		if p.Status.Phase == "Failed" {
			return false, "", fmt.Errorf("mirrored image %s failed to run: %s", okImage, status)
		}
		return p.Status.Phase == "Succeeded", status, nil
	})
	if err != nil {
		return fmt.Errorf("mirrored image %s did not run: %w", okImage, err)
	}
	l.logf("PASS  mirrored image runs:            %s\n", okImage)

	// 2. Every un-mirrored public image must fail to pull.
	for i, b := range blocked {
		var reason string
		err = kube.Poll(ctx, 3*time.Minute, 2*time.Second, func(ctx context.Context) (bool, string, error) {
			p, status, err := l.probePod(ctx, blockedPodName(i))
			if err != nil || p == nil {
				return false, status, err
			}
			if p.Status.Phase == "Succeeded" || p.Status.Phase == "Running" {
				return false, "", fmt.Errorf("AIR GAP LEAK: un-mirrored image %s was pulled and ran", b)
			}
			for _, cs := range p.Status.ContainerStatuses {
				if w := cs.State.Waiting; w != nil && (w.Reason == "ErrImagePull" || w.Reason == "ImagePullBackOff") {
					reason = w.Reason + ": " + firstLine(w.Message)
					return true, status, nil
				}
			}
			return false, status, nil
		})
		if err != nil {
			return fmt.Errorf("un-mirrored image %s: %w", b, err)
		}
		l.logf("PASS  un-mirrored image cannot pull:  %s\n      %s\n", b, reason)
	}
	return nil
}

func blockedPodName(i int) string { return fmt.Sprintf("not-mirrored-%d", i) }

func (l *Lab) probePod(ctx context.Context, name string) (*kube.Pod, string, error) {
	pods, err := l.Kube().Pods(ctx, VerifyNamespace, "")
	if err != nil {
		return nil, "", err
	}
	for i := range pods {
		if pods[i].Metadata.Name != name {
			continue
		}
		p := &pods[i]
		status := "phase=" + p.Status.Phase
		for _, cs := range p.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				status += " waiting=" + cs.State.Waiting.Reason
			}
		}
		return p, status, nil
	}
	return nil, "pod " + name + " not created yet", nil
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 300 {
		s = s[:300] + "..."
	}
	return s
}
