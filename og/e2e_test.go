//go:build e2e

// Package og's end-to-end tests exercise the full path against a real registry
// (zot) using the real oras CLI to publish artifacts.
//
// They are gated behind the "e2e" build tag so the default `go test ./...` stays
// hermetic. Run them with:
//
//	go test -tags e2e ./...
//
// The registry address defaults to "registry:5000" (the devcontainer's zot) and
// can be overridden with ORAS_GET_E2E_REGISTRY. The tests skip themselves if the
// oras CLI is missing or the registry is unreachable.
package og_test

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lesomnus/oras-get/match"
	"github.com/lesomnus/oras-get/og"
	"github.com/lesomnus/oras-get/og/upstream"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"
)

func e2eRegistry(t *testing.T) string {
	t.Helper()

	addr := os.Getenv("ORAS_GET_E2E_REGISTRY")
	if addr == "" {
		addr = "registry:5000"
	}
	if _, err := exec.LookPath("oras"); err != nil {
		t.Skipf("oras CLI not found: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/v2/", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("registry %q unreachable: %v", addr, err)
	}
	res.Body.Close()
	return addr
}

// oras runs an oras subcommand against a plain-http registry and fails the test
// on error. --plain-http is appended so it attaches to the subcommand rather
// than being read as a (nonexistent) global flag.
func oras(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("oras", append(args, "--plain-http")...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "ORAS_EXPERIMENTAL=true")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("oras %v failed: %v\n%s", args, err, out)
	}
}

func e2eRouter(t *testing.T, addr string) *og.Router {
	t.Helper()
	r, err := remote.NewRegistry(addr)
	require.NoError(t, err)
	r.PlainHTTP = true

	return &og.Router{
		Upstreams: map[string]upstream.Upstream{
			"z": {Scheme: "http", Domain: addr, Registry: r},
		},
		Matchers: []match.Matcher{match.FixedMatcher("z")},
	}
}

func TestE2EMultiFile(t *testing.T) {
	addr := e2eRegistry(t)
	router := e2eRouter(t, addr)

	// Publish a multi-file artifact with a nested path.
	dir := t.TempDir()
	tool := []byte("#!/bin/sh\necho hello\n")
	readme := []byte("# oras-get e2e\n")
	nested := []byte("nested payload")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tool"), tool, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), readme, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "share"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "share", "data.bin"), nested, 0o644))

	repo := addr + "/oras-get-e2e/bundle:v1"
	oras(t, dir, "push", "--disable-path-validation", repo, "tool", "README.md", "share/data.bin")

	t.Run("fetch each file by path", func(t *testing.T) {
		for name, want := range map[string][]byte{
			"tool":           tool,
			"README.md":      readme,
			"share/data.bin": nested,
		} {
			x := require.New(t)
			res, body := do(t, router, http.MethodGet, "/oras-get-e2e/bundle:v1/"+name)
			x.Equal(http.StatusOK, res.StatusCode, "%s: %s", name, body)
			x.Equal(want, body, "content of %s", name)
		}
	})

	t.Run("400 without a file path", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/oras-get-e2e/bundle:v1")
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("404 for an unknown file", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/oras-get-e2e/bundle:v1/missing")
		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("list files", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/oras-get-e2e/bundle:v1/")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Contains(string(body), "tool")
		x.Contains(string(body), "README.md")
		x.Contains(string(body), "share/data.bin")
	})

	t.Run("list tags", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/oras-get-e2e/bundle:_")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Contains(string(body), "v1")
	})
}

func TestE2EMultiArch(t *testing.T) {
	addr := e2eRegistry(t)
	router := e2eRouter(t, addr)

	dir := t.TempDir()
	amd := []byte("amd64 binary")
	arm := []byte("arm64 binary")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tool"), amd, 0o644))
	base := addr + "/oras-get-e2e/multiarch"
	oras(t, dir, "push", "--artifact-platform", "linux/amd64", base+":amd64", "tool")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "tool"), arm, 0o644))
	oras(t, dir, "push", "--artifact-platform", "linux/arm64", base+":arm64", "tool")

	// Combine the two platform manifests into an index tagged v1.
	oras(t, dir, "manifest", "index", "create", base+":v1", "amd64", "arm64")

	t.Run("fetch a file per platform", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/oras-get-e2e/multiarch:v1/tool?platform=linux/amd64")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(amd, body)

		res, body = do(t, router, http.MethodGet, "/oras-get-e2e/multiarch:v1/tool?platform=linux/arm64")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(arm, body)
	})

	t.Run("400 without a platform", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/oras-get-e2e/multiarch:v1/tool")
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}
