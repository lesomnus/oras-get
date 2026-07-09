package og_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/lesomnus/oras-get/internal/registrytest"
	"github.com/lesomnus/oras-get/match"
	"github.com/lesomnus/oras-get/og"
	"github.com/lesomnus/oras-get/og/upstream"
	"github.com/lesomnus/otx/log"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"
)

func newRouter(t *testing.T, reg *registrytest.Registry) *og.Router {
	t.Helper()
	x := require.New(t)

	r, err := remote.NewRegistry(reg.Host())
	x.NoError(err)
	r.PlainHTTP = true
	r.Client = reg.Client()

	return &og.Router{
		Upstreams: map[string]upstream.Upstream{
			"test": {Scheme: "http", Domain: reg.Host(), Registry: r},
		},
		Matchers: []match.Matcher{match.FixedMatcher("test")},
	}
}

func do(t *testing.T, router *og.Router, method, target string) (*http.Response, []byte) {
	return doH(t, router, method, target, nil)
}

func doH(t *testing.T, router *og.Router, method, target string, header http.Header) (*http.Response, []byte) {
	t.Helper()
	ctx := log.Into(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(method, target, nil).WithContext(ctx)
	for k, vs := range header {
		req.Header[k] = vs
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	return res, body
}

func TestRouter(t *testing.T) {
	reg := registrytest.New(t)

	// A multi-file artifact.
	toolData := []byte("#!/bin/sh\necho tool\n")
	readmeData := []byte("# readme\n")
	app := reg.AddManifest([]oci.Descriptor{
		reg.AddBlob("application/x-executable", "bin/tool", toolData),
		reg.AddBlob("text/markdown", "README.md", readmeData),
	}, nil)
	reg.Tag("app", "v1", app)

	// A single-file artifact.
	soloData := []byte("i am alone")
	solo := reg.AddManifest([]oci.Descriptor{
		reg.AddBlob("application/octet-stream", "solo.bin", soloData),
	}, nil)
	reg.Tag("solo", "v1", solo)

	// A multi-arch index of multi-file manifests.
	amdData := []byte("amd64 tool")
	armData := []byte("arm64 tool")
	idx := reg.AddIndex([]oci.Descriptor{
		reg.AddManifest([]oci.Descriptor{reg.AddBlob("application/x-executable", "bin/tool", amdData)},
			&oci.Platform{OS: "linux", Architecture: "amd64"}),
		reg.AddManifest([]oci.Descriptor{reg.AddBlob("application/x-executable", "bin/tool", armData)},
			&oci.Platform{OS: "linux", Architecture: "arm64"}),
	})
	reg.Tag("multi", "v1", idx)

	// A multi-file artifact where a file name contains a colon.
	verData := []byte("v1.2 payload")
	colon := reg.AddManifest([]oci.Descriptor{
		reg.AddBlob("text/plain", "ver:1.2.txt", verData),
		reg.AddBlob("text/plain", "other.txt", []byte("other")),
	}, nil)
	reg.Tag("withcolon", "v1", colon)

	router := newRouter(t, reg)

	t.Run("fetch a file whose name contains a colon", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/withcolon:v1/ver:1.2.txt")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(verData, body)
	})

	t.Run("platform arch is normalized (x86_64 -> amd64)", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/multi:v1/bin/tool?platform=linux/x86_64")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(amdData, body)
	})

	t.Run("400 for a platform without an arch", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/multi:v1/bin/tool?platform=linux")
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("fetch a file by path", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/app:v1/bin/tool")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(toolData, body)
		x.Equal("application/x-executable", res.Header.Get("Content-Type"))

		res, body = do(t, router, http.MethodGet, "/app:v1/README.md")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(readmeData, body)
	})

	t.Run("HEAD returns headers without a body", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodHead, "/app:v1/bin/tool")
		x.Equal(http.StatusOK, res.StatusCode)
		x.Empty(body)
		x.Equal("application/x-executable", res.Header.Get("Content-Type"))
	})

	t.Run("404 for an unknown file", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/app:v1/nope")
		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("400 when a file must be chosen", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/app:v1")
		x.Equal(http.StatusBadRequest, res.StatusCode)
		x.Contains(string(body), "bin/tool")
		x.Contains(string(body), "README.md")
	})

	t.Run("list files with a trailing slash", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/app:v1/")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Contains(string(body), "bin/tool")
		x.Contains(string(body), "README.md")
	})

	t.Run("single-file artifact needs no path", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/solo:v1")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(soloData, body)
	})

	t.Run("select a manifest by platform", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/multi:v1/bin/tool?platform=linux/amd64")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(amdData, body)

		res, body = do(t, router, http.MethodGet, "/multi:v1/bin/tool?platform=linux/arm64")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(armData, body)
	})

	t.Run("400 when an index is fetched without a platform", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/multi:v1/bin/tool")
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("400 when a platform is given for a plain manifest", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/app:v1/bin/tool?platform=linux/amd64")
		require.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("404 for an absent platform", func(t *testing.T) {
		res, _ := do(t, router, http.MethodGet, "/multi:v1/bin/tool?platform=linux/ppc64le")
		require.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("list tags", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/app:_")
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Contains(string(body), "v1")
	})
}

func TestRouterCaching(t *testing.T) {
	reg := registrytest.New(t)

	data := []byte("cache me maybe")
	blob := reg.AddBlob("application/octet-stream", "file.bin", data)
	reg.Tag("cache", "v1", reg.AddManifest([]oci.Descriptor{blob}, nil))
	router := newRouter(t, reg)

	etag := `"` + blob.Digest.String() + `"`

	t.Run("GET carries an ETag and Cache-Control", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodGet, "/cache:v1/file.bin")
		x.Equal(http.StatusOK, res.StatusCode)
		x.Equal(data, body)
		x.Equal(etag, res.Header.Get("ETag"))
		x.NotEmpty(res.Header.Get("Cache-Control"))
	})

	t.Run("HEAD returns metadata without a body", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodHead, "/cache:v1/file.bin")
		x.Equal(http.StatusOK, res.StatusCode)
		x.Empty(body)
		x.Equal(etag, res.Header.Get("ETag"))
		x.Equal(strconv.Itoa(len(data)), res.Header.Get("Content-Length"))
	})

	t.Run("a matching If-None-Match yields 304 with no body", func(t *testing.T) {
		x := require.New(t)
		res, body := doH(t, router, http.MethodGet, "/cache:v1/file.bin", http.Header{"If-None-Match": {etag}})
		x.Equal(http.StatusNotModified, res.StatusCode)
		x.Empty(body)
		x.Equal(etag, res.Header.Get("ETag"))
	})

	t.Run("a stale If-None-Match still serves the content", func(t *testing.T) {
		x := require.New(t)
		res, body := doH(t, router, http.MethodGet, "/cache:v1/file.bin", http.Header{"If-None-Match": {`"sha256:0000"`}})
		x.Equal(http.StatusOK, res.StatusCode)
		x.Equal(data, body)
	})

	t.Run("HEAD on a tag list is allowed", func(t *testing.T) {
		x := require.New(t)
		res, body := do(t, router, http.MethodHead, "/cache:_")
		x.Equal(http.StatusOK, res.StatusCode)
		x.Empty(body)
	})
}

func TestRouterRedirect(t *testing.T) {
	reg := registrytest.New(t)

	data := []byte("redirect me")
	blob := reg.AddBlob("application/octet-stream", "data.bin", data)
	reg.Tag("redir", "v1", reg.AddManifest([]oci.Descriptor{blob}, nil))

	r, err := remote.NewRegistry(reg.Host())
	require.NoError(t, err)
	r.PlainHTTP = true
	r.Client = reg.Client()

	router := &og.Router{
		Upstreams: map[string]upstream.Upstream{
			"test": {Scheme: "http", Domain: reg.Host(), Registry: r, Redirect: true},
		},
		Matchers: []match.Matcher{match.FixedMatcher("test")},
	}

	t.Run("a blob request redirects to the upstream blob URL", func(t *testing.T) {
		x := require.New(t)
		res, _ := do(t, router, http.MethodGet, "/redir:v1/data.bin")
		x.Equal(http.StatusTemporaryRedirect, res.StatusCode)
		x.Equal("http://"+reg.Host()+"/v2/redir/blobs/"+blob.Digest.String(), res.Header.Get("Location"))
	})

	t.Run("a tag listing redirects to the upstream tags URL", func(t *testing.T) {
		x := require.New(t)
		res, _ := do(t, router, http.MethodGet, "/redir:_")
		x.Equal(http.StatusTemporaryRedirect, res.StatusCode)
		x.Equal("http://"+reg.Host()+"/v2/redir/tags/list", res.Header.Get("Location"))
	})

	t.Run("a matching If-None-Match yields 304 instead of a redirect", func(t *testing.T) {
		x := require.New(t)
		etag := `"` + blob.Digest.String() + `"`
		res, _ := doH(t, router, http.MethodGet, "/redir:v1/data.bin", http.Header{"If-None-Match": {etag}})
		x.Equal(http.StatusNotModified, res.StatusCode)
	})
}
