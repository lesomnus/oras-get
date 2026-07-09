package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lesomnus/oras-get/og/handler"
	"github.com/lesomnus/oras-get/og/upstream"
	"github.com/lesomnus/oras-get/refs"
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"
)

// titled builds a layer descriptor for data with the given file title.
func titled(mediaType, title string, data []byte) oci.Descriptor {
	return oci.Descriptor{
		MediaType:   mediaType,
		Size:        int64(len(data)),
		Digest:      digest.FromBytes(data),
		Annotations: map[string]string{oci.AnnotationTitle: title},
	}
}

// blobServer serves each blob at /v2/foo/blobs/<digest> and returns a repo
// bound to it.
func blobServer(t *testing.T, blobs map[digest.Digest][]byte) upstream.Repository {
	t.Helper()
	x := require.New(t)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for d, data := range blobs {
			if strings.HasSuffix(r.URL.Path, "/blobs/"+d.String()) {
				w.Write(data)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(s.Close)

	host := s.URL[len("http://"):]
	reg, err := remote.NewRegistry(host)
	x.NoError(err)
	reg.Client = s.Client()
	reg.PlainHTTP = true

	ref, err := refs.Parse(host + "/foo:bar")
	x.NoError(err)

	repo, err := upstream.Upstream{Registry: reg}.Repository(t.Context(), ref)
	x.NoError(err)
	return repo
}

// serve resolves and serves a manifest handler for the given manifest. refSuffix
// is appended to the reference "foo:bar" verbatim, e.g. "/bin/tool" to select a
// file or "/" to request a listing. It returns the recorded response.
func serve(t *testing.T, repo upstream.Repository, refSuffix string, m oci.Manifest) *http.Response {
	t.Helper()
	x := require.New(t)

	name := "foo:bar" + refSuffix
	if d := repo.Reference.Domain(); d != "" {
		name = d + "/" + name
	}
	ref, err := refs.Parse(name)
	x.NoError(err)
	repo.Reference = refs.WithDomain(ref, "")

	h, ok := handler.Resolve(repo, oci.Descriptor{MediaType: oci.MediaTypeImageManifest})
	x.True(ok)

	raw, err := json.Marshal(m)
	x.NoError(err)
	x.NoError(h.Parse(bytes.NewReader(raw)))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(w, r)
	return w.Result()
}

func TestManifest(t *testing.T) {
	t.Run("retrieve a single-file blob without a path", func(t *testing.T) {
		data := []byte("Royale with Cheese")
		dgst := digest.FromBytes(data)

		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{dgst: data})

		res := serve(t, repo, "", oci.Manifest{
			Layers: []oci.Descriptor{{MediaType: "application/foo", Size: int64(len(data)), Digest: dgst}},
		})

		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(data, body)
		x.Equal("application/foo", res.Header.Get("Content-Type"))
		x.Equal(fmt.Sprintf("%d", len(data)), res.Header.Get("Content-Length"))
	})

	t.Run("select a file by its title", func(t *testing.T) {
		a := []byte("i am tool")
		b := []byte("i am the readme")
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{
			digest.FromBytes(a): a,
			digest.FromBytes(b): b,
		})

		m := oci.Manifest{Layers: []oci.Descriptor{
			titled("application/x-tool", "bin/tool", a),
			titled("text/markdown", "README.md", b),
		}}

		res := serve(t, repo, "/bin/tool", m)
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(a, body)
		x.Equal("application/x-tool", res.Header.Get("Content-Type"))

		res = serve(t, repo, "/README.md", m)
		body, err = io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(b, body)
	})

	t.Run("select a file with an equivalent path", func(t *testing.T) {
		a := []byte("i am tool")
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{digest.FromBytes(a): a})

		m := oci.Manifest{Layers: []oci.Descriptor{
			titled("application/x-tool", "bin/tool", a),
			titled("text/plain", "note.txt", []byte("x")),
		}}

		// "./bin/tool" normalizes to the same file as "bin/tool".
		res := serve(t, repo, "/./bin/tool", m)
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(a, body)
	})

	t.Run("404 if the requested file is not present", func(t *testing.T) {
		a := []byte("i am tool")
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{digest.FromBytes(a): a})

		res := serve(t, repo, "/does/not/exist", oci.Manifest{Layers: []oci.Descriptor{
			titled("application/x-tool", "bin/tool", a),
		}})
		x.Equal(http.StatusNotFound, res.StatusCode)
	})

	t.Run("list files on a trailing slash", func(t *testing.T) {
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{})

		res := serve(t, repo, "/", oci.Manifest{Layers: []oci.Descriptor{
			titled("application/x-tool", "bin/tool", []byte("a")),
			titled("text/markdown", "README.md", []byte("b")),
		}})
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		lines := strings.Fields(string(body))
		x.ElementsMatch([]string{"bin/tool", "README.md"}, lines)
	})

	t.Run("400 if multiple files and none is specified", func(t *testing.T) {
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{})

		res := serve(t, repo, "", oci.Manifest{Layers: []oci.Descriptor{
			titled("application/x-tool", "bin/tool", []byte("a")),
			titled("text/markdown", "README.md", []byte("b")),
		}})
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusBadRequest, res.StatusCode)
		// The message lists the available files.
		x.Contains(string(body), "bin/tool")
		x.Contains(string(body), "README.md")
	})

	t.Run("412 if the blob is not found upstream", func(t *testing.T) {
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{}) // serves 404 for any blob

		res := serve(t, repo, "", oci.Manifest{Layers: []oci.Descriptor{{}}})
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusPreconditionFailed, res.StatusCode, string(body))
	})

	t.Run("412 if the manifest has no layers", func(t *testing.T) {
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{})

		res := serve(t, repo, "", oci.Manifest{Layers: []oci.Descriptor{}})
		x.Equal(http.StatusPreconditionFailed, res.StatusCode)
	})

	t.Run("distinctly titled layers are each reachable by exact title", func(t *testing.T) {
		a := []byte("plain tool")
		b := []byte("dotted tool")
		x := require.New(t)
		repo := blobServer(t, map[digest.Digest][]byte{
			digest.FromBytes(a): a,
			digest.FromBytes(b): b,
		})

		// Two layers whose titles normalize to the same path but differ exactly.
		m := oci.Manifest{Layers: []oci.Descriptor{
			titled("application/x-tool", "bin/tool", a),
			titled("application/x-tool", "./bin/tool", b),
		}}

		res := serve(t, repo, "/bin/tool", m)
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(a, body)

		res = serve(t, repo, "/./bin/tool", m)
		body, err = io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(b, body)
	})

	t.Run("untitled layers are omitted from the listing", func(t *testing.T) {
		x := require.New(t)
		named := []byte("readme")
		repo := blobServer(t, map[digest.Digest][]byte{})

		// One titled layer and one anonymous layer (no title annotation).
		anon := oci.Descriptor{MediaType: "application/octet-stream", Digest: digest.FromBytes([]byte("anon")), Size: 4}
		res := serve(t, repo, "/", oci.Manifest{Layers: []oci.Descriptor{
			titled("text/markdown", "README.md", named),
			anon,
		}})
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal([]string{"README.md"}, strings.Fields(string(body)))
		x.NotContains(string(body), anon.Digest.String())
	})
}
