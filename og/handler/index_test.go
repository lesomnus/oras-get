package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lesomnus/oras-get/og/handler"
	"github.com/lesomnus/oras-get/og/upstream"
	"github.com/lesomnus/oras-get/refs"
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"
)

func TestIndex(t *testing.T) {
	t.Run("retrieve a blob with given platform", func(t *testing.T) {
		index := oci.Index{
			Manifests: []oci.Descriptor{},
		}
		manifests := map[string]*struct {
			v    oci.Manifest
			data []byte
		}{
			"amd64": {
				v: oci.Manifest{
					Layers: []oci.Descriptor{
						{
							MediaType: "application/foo",
							Digest:    "alg:xxx",
						},
					},
				},
			},
			"arm64": {
				v: oci.Manifest{
					Layers: []oci.Descriptor{
						{
							MediaType: "application/bar",
							Digest:    "alg:yyy",
						},
					},
				},
			},
		}
		for k, manifest := range manifests {
			data, _ := json.Marshal(manifest.v)
			manifest.data = data

			index.Manifests = append(index.Manifests, oci.Descriptor{
				MediaType: oci.MediaTypeImageManifest,
				Size:      int64(len(data)),
				Digest:    digest.Digest("alg:" + k),
				Platform: &oci.Platform{
					OS:           "linux",
					Architecture: k,
				},
			})
		}

		x := require.New(t)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v2/foo/manifests/alg:amd64":
				manifest := manifests["amd64"]
				w.Header().Set("Content-Type", oci.MediaTypeImageManifest)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(manifest.data)))
				w.Write(manifest.data)

			case "/v2/foo/manifests/alg:arm64":
				manifest := manifests["arm64"]
				w.Header().Set("Content-Type", oci.MediaTypeImageManifest)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(manifest.data)))
				w.Write(manifest.data)
			}
		}))
		defer s.Close()

		ref, err := refs.Parse(s.URL[len("http://"):] + "/foo:bar")
		x.NoError(err)

		reg, err := remote.NewRegistry(ref.Domain())
		x.NoError(err)

		reg.Client = s.Client()
		reg.PlainHTTP = true

		m, err := json.Marshal(index)
		x.NoError(err)

		for p, manifest := range manifests {
			upstream := upstream.Upstream{Registry: reg}
			repo, err := upstream.Repository(t.Context(), refs.WithPlatform(ref, refs.Platform("linux/"+p)))

			h, ok := handler.Resolve(repo, oci.Descriptor{MediaType: oci.MediaTypeImageIndex})
			x.True(ok)

			err = h.Parse(bytes.NewReader(m))
			x.NoError(err)

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			h.ServeHTTP(w, r)

			res := w.Result()
			body, err := io.ReadAll(res.Body)
			x.NoError(err)
			x.Equal(http.StatusOK, res.StatusCode, string(body))
			x.Equal(manifest.v.Layers[0].MediaType, res.Header.Get("Content-Type"))
		}
	})
	t.Run("404 if no manifest for the specified platform", func(t *testing.T) {
		index := oci.Index{
			Manifests: []oci.Descriptor{
				{
					MediaType: oci.MediaTypeImageManifest,
					Size:      123,
					Digest:    "alg:xxx",
					Platform: &oci.Platform{
						OS:           "linux",
						Architecture: "amd64",
					},
				},
			},
		}

		x := require.New(t)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("unreachable")
		}))
		defer s.Close()

		ref, err := refs.Parse(s.URL[len("http://"):] + "/foo:bar")
		x.NoError(err)

		reg, err := remote.NewRegistry(ref.Domain())
		x.NoError(err)

		reg.Client = s.Client()
		reg.PlainHTTP = true

		upstream := upstream.Upstream{Registry: reg}
		repo, err := upstream.Repository(t.Context(), refs.WithPlatform(ref, refs.Platform("linux/arm64")))

		m, err := json.Marshal(index)
		x.NoError(err)

		h, ok := handler.Resolve(repo, oci.Descriptor{MediaType: oci.MediaTypeImageIndex})
		x.True(ok)

		err = h.Parse(bytes.NewReader(m))
		x.NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(w, r)

		res := w.Result()
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusNotFound, res.StatusCode, string(body))
	})
}
