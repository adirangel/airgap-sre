package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/adirangel/airgap-sre/backend/internal/config"
	"github.com/adirangel/airgap-sre/backend/internal/lab"
	"github.com/adirangel/airgap-sre/backend/internal/toolchain"
)

// newLab loads the pinned config and builds a Lab bound to cmd's output.
func (a *app) newLab(cmd *cobra.Command) (*lab.Lab, error) {
	p, err := a.paths()
	if err != nil {
		return nil, err
	}
	versions, err := config.LoadVersions(p.Infra("versions.yaml"))
	if err != nil {
		return nil, err
	}
	images, err := config.LoadImages(p.Infra("images.yaml"))
	if err != nil {
		return nil, err
	}
	out := cmd.OutOrStdout()
	return &lab.Lab{
		Paths:  p,
		Cfg:    versions.Lab,
		Images: images,
		TC:     toolchain.New(p, versions, a.runner, out),
		Out:    out,
	}, nil
}

func newToolsCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{Use: "tools", Short: "Manage the pinned kind/kubectl/helm binaries in .tools/"}
	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Download and checksum-verify kind, kubectl and helm into .tools/",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			bins, err := l.TC.InstallAll(cmd.Context())
			if err != nil {
				return err
			}
			names := make([]string, 0, len(bins))
			for n := range bins {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%-8s %s\n", n, bins[n])
			}
			return nil
		},
	})
	return cmd
}

func newLabCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{Use: "lab", Short: "Create, inspect and remove the local lab (kind cluster + registry)"}

	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Start the registry and create the kind cluster (idempotent)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			return l.Up(cmd.Context())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "down",
		Short: "Delete the kind cluster and the registry container (mirrored images are kept in the volume)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			return l.Down(cmd.Context())
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show the state of the registry and the cluster",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			st, err := l.Status(cmd.Context())
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "registry  %s: %s\n", l.Cfg.Registry.ContainerName, state(st.RegistryExists, st.RegistryRunning))
			if st.RegistryRunning {
				_, _ = fmt.Fprintf(out, "          %s, %d mirrored repositories\n", l.Cfg.Registry.HostAddr(), len(st.Repositories))
				for _, r := range st.Repositories {
					_, _ = fmt.Fprintf(out, "            %s\n", r)
				}
			}
			_, _ = fmt.Fprintf(out, "cluster   %s: %s\n", l.Cfg.ClusterName, state(st.ClusterExists, st.ClusterExists))
			for _, n := range st.Nodes {
				ready := "NotReady"
				if n.Ready {
					ready = "Ready"
				}
				_, _ = fmt.Fprintf(out, "          node %s: %s\n", n.Name, ready)
			}
			if st.ClusterExists {
				_, _ = fmt.Fprintf(out, "kubeconfig %s\n", l.Paths.Kubeconfig())
			}
			return nil
		},
	})

	var group, arch string
	mirror := &cobra.Command{
		Use:   "mirror",
		Short: "Copy the images listed in infrastructure/images.yaml into the lab registry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			return l.MirrorGroup(cmd.Context(), group, arch)
		},
	}
	mirror.Flags().StringVar(&group, "group", "all", "image group to mirror (see images.yaml)")
	mirror.Flags().StringVar(&arch, "arch", "", "CPU architecture to mirror (default: this machine's)")
	cmd.AddCommand(mirror)

	cmd.AddCommand(&cobra.Command{
		Use:   "verify",
		Short: "Prove the air gap: a mirrored image runs, an un-mirrored public image cannot be pulled",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			l, err := a.newLab(cmd)
			if err != nil {
				return err
			}
			return l.Verify(cmd.Context())
		},
	})
	return cmd
}

func state(exists, running bool) string {
	switch {
	case running:
		return "running"
	case exists:
		return "stopped"
	default:
		return "absent"
	}
}
