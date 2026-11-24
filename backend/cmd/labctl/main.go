// labctl is the single cross-platform CLI for the airgap-sre lab: tools,
// cluster, registry, scenarios, evals and cleanup.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/adirangel/airgap-sre/backend/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cli.NewRoot().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
