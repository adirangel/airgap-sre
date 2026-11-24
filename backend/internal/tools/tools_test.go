package tools

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adirangel/airgap-sre/backend/internal/config"
)

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func TestExpand(t *testing.T) {
	tool := config.Tool{Name: "helm", Version: "v1.2.3"}
	win := Platform{OS: "windows", Arch: "amd64"}
	lin := Platform{OS: "linux", Arch: "arm64"}
	const tmpl = "https://x/helm-{version}-{os}-{arch}.{archive}#{exe}"
	if got, want := Expand(tmpl, tool, win), "https://x/helm-v1.2.3-windows-amd64.zip#.exe"; got != want {
		t.Errorf("windows: got %s want %s", got, want)
	}
	if got, want := Expand(tmpl, tool, lin), "https://x/helm-v1.2.3-linux-arm64.tar.gz#"; got != want {
		t.Errorf("linux: got %s want %s", got, want)
	}
}

func TestEnsureBinary(t *testing.T) {
	payload := []byte("fake kind binary")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(payload) }))
	defer srv.Close()

	p := Platform{OS: "linux", Arch: "amd64"}
	tool := config.Tool{Name: "kind", Version: "v0.0.1", URL: srv.URL + "/kind-{os}-{arch}", SHA256: map[string]string{p.String(): sum(payload)}}
	in := Installer{ToolsDir: t.TempDir(), Platform: p}

	path, err := in.Ensure(context.Background(), tool)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(in.ToolsDir, "kind-v0.0.1", "kind"); path != want {
		t.Errorf("path = %s, want %s", path, want)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("installed file wrong: %v", err)
	}

	// Second call must not hit the network.
	srv.Close()
	if _, err := in.Ensure(context.Background(), tool); err != nil {
		t.Fatalf("second Ensure should reuse the binary: %v", err)
	}
}

func TestEnsureRejectsBadChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("tampered")) }))
	defer srv.Close()

	p := Platform{OS: "linux", Arch: "amd64"}
	tool := config.Tool{Name: "kind", Version: "v0.0.1", URL: srv.URL, SHA256: map[string]string{p.String(): sum([]byte("original"))}}
	in := Installer{ToolsDir: t.TempDir(), Platform: p}

	_, err := in.Ensure(context.Background(), tool)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("want checksum mismatch, got %v", err)
	}
	if _, statErr := os.Stat(BinaryPath(in.ToolsDir, tool, p)); !os.IsNotExist(statErr) {
		t.Fatal("a binary with a bad checksum was left on disk")
	}
	entries, _ := os.ReadDir(filepath.Join(in.ToolsDir, "kind-v0.0.1"))
	if len(entries) != 0 {
		t.Fatalf("temporary download left behind: %v", entries)
	}
}

func TestEnsureMissingPlatformPin(t *testing.T) {
	tool := config.Tool{Name: "kind", Version: "v0.0.1", URL: "http://unused", SHA256: map[string]string{"linux/amd64": sum(nil)}}
	in := Installer{ToolsDir: t.TempDir(), Platform: Platform{OS: "plan9", Arch: "mips"}}
	if _, err := in.Ensure(context.Background(), tool); err == nil {
		t.Fatal("expected an error for an unpinned platform")
	}
}

func TestEnsureArchives(t *testing.T) {
	binary := []byte("fake helm binary")

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	f, _ := zw.Create("windows-amd64/helm.exe")
	_, _ = f.Write(binary)
	_ = zw.Close()

	var tgzBuf bytes.Buffer
	gz := gzip.NewWriter(&tgzBuf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "linux-amd64/README", Mode: 0o644, Size: 2})
	_, _ = tw.Write([]byte("hi"))
	_ = tw.WriteHeader(&tar.Header{Name: "linux-amd64/helm", Mode: 0o755, Size: int64(len(binary))})
	_, _ = tw.Write(binary)
	_ = tw.Close()
	_ = gz.Close()

	cases := []struct {
		platform Platform
		archive  []byte
	}{
		{Platform{OS: "windows", Arch: "amd64"}, zipBuf.Bytes()},
		{Platform{OS: "linux", Arch: "amd64"}, tgzBuf.Bytes()},
	}
	for _, c := range cases {
		t.Run(c.platform.String(), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(c.archive) }))
			defer srv.Close()
			tool := config.Tool{
				Name: "helm", Version: "v9.9.9", URL: srv.URL + "/helm.{archive}",
				ArchivePath: "{os}-{arch}/helm{exe}",
				SHA256:      map[string]string{c.platform.String(): sum(c.archive)},
			}
			in := Installer{ToolsDir: t.TempDir(), Platform: c.platform}
			path, err := in.Ensure(context.Background(), tool)
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, binary) {
				t.Fatalf("extracted binary wrong (err=%v)", err)
			}
		})
	}
}
