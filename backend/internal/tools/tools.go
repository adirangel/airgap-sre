// Package tools downloads the pinned kind/kubectl/helm binaries into .tools/
// and verifies their checksums. Nothing is installed outside the repository.
package tools

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/adirangel/airgap-sre/backend/internal/config"
)

// Platform is a GOOS/GOARCH pair.
type Platform struct {
	OS   string
	Arch string
}

// Host is the platform labctl is running on.
func Host() Platform { return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH} }

func (p Platform) String() string { return p.OS + "/" + p.Arch }

func (p Platform) exe() string {
	if p.OS == "windows" {
		return ".exe"
	}
	return ""
}

func (p Platform) archive() string {
	if p.OS == "windows" {
		return "zip"
	}
	return "tar.gz"
}

// Expand fills the {version} {os} {arch} {exe} {archive} placeholders.
func Expand(tmpl string, t config.Tool, p Platform) string {
	return strings.NewReplacer(
		"{version}", t.Version,
		"{os}", p.OS,
		"{arch}", p.Arch,
		"{exe}", p.exe(),
		"{archive}", p.archive(),
	).Replace(tmpl)
}

// BinaryPath is where the tool lives once installed:
// .tools/<name>-<version>/<name>[.exe]. The version is part of the path, so
// bumping a pin never reuses an old binary.
func BinaryPath(toolsDir string, t config.Tool, p Platform) string {
	return filepath.Join(toolsDir, t.Name+"-"+t.Version, t.Name+p.exe())
}

// Installer downloads tools.
type Installer struct {
	ToolsDir string
	Platform Platform
	Client   *http.Client
	Log      io.Writer
}

// Ensure returns the path of the tool binary, downloading and verifying it
// first if it is not there yet.
func (in Installer) Ensure(ctx context.Context, t config.Tool) (string, error) {
	dest := BinaryPath(in.ToolsDir, t, in.Platform)
	if _, err := os.Stat(dest); err == nil {
		return dest, nil
	}
	want, ok := t.SHA256[in.Platform.String()]
	if !ok {
		return "", fmt.Errorf("%s %s has no pinned checksum for %s in versions.yaml", t.Name, t.Version, in.Platform)
	}
	url := Expand(t.URL, t, in.Platform)
	in.logf("downloading %s %s for %s\n  %s\n", t.Name, t.Version, in.Platform, url)

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), "download-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	got, err := in.download(ctx, url, tmp)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	if got != want {
		return "", fmt.Errorf("checksum mismatch for %s: got %s, pinned %s — refusing to install", url, got, want)
	}
	in.logf("  sha256 ok (%s)\n", got)

	if t.ArchivePath == "" {
		if err := os.Rename(tmpName, dest); err != nil {
			return "", err
		}
	} else {
		member := Expand(t.ArchivePath, t, in.Platform)
		if err := extract(tmpName, in.Platform.archive(), member, dest); err != nil {
			return "", fmt.Errorf("extract %s from %s: %w", member, url, err)
		}
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return "", err
	}
	return dest, nil
}

func (in Installer) logf(format string, args ...any) {
	if in.Log != nil {
		_, _ = fmt.Fprintf(in.Log, format, args...)
	}
}

func (in Installer) download(ctx context.Context, url string, w io.Writer) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	client := in.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %s", resp.Status)
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(w, h), resp.Body); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extract copies one member out of a zip or tar.gz archive to dest.
func extract(archivePath, kind, member, dest string) error {
	switch kind {
	case "zip":
		zr, err := zip.OpenReader(archivePath)
		if err != nil {
			return err
		}
		defer func() { _ = zr.Close() }()
		for _, f := range zr.File {
			if f.Name != member {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()
			return writeFile(dest, rc)
		}
	case "tar.gz":
		f, err := os.Open(archivePath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		gz, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer func() { _ = gz.Close() }()
		tr := tar.NewReader(gz)
		for {
			hdr, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			if hdr.Name == member {
				return writeFile(dest, tr)
			}
		}
	default:
		return fmt.Errorf("unknown archive type %q", kind)
	}
	return fmt.Errorf("member %q not found in archive", member)
}

func writeFile(dest string, r io.Reader) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, r); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
