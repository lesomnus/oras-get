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
	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/registry/remote"
)

func TestManifest(t *testing.T) {
	t.Run("retrieve a blob", func(t *testing.T) {
		data := []byte("Royale with Cheese")
		digest := digest.FromBytes(data)

		x := require.New(t)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(data)
		}))
		defer s.Close()

		repo, err := remote.NewRepository(fmt.Sprintf("%s/foo", s.URL[len("http://"):]))
		x.NoError(err)

		repo.Client = s.Client()
		repo.PlainHTTP = true
		h, ok := handler.Resolve(repo, oci.Descriptor{MediaType: oci.MediaTypeImageManifest}, "")
		x.True(ok)

		m, err := json.Marshal(oci.Manifest{
			Layers: []oci.Descriptor{
				{
					MediaType: "application/foo",
					Size:      int64(len(data)),
					Digest:    digest,
				},
			},
		})
		x.NoError(err)

		err = h.Parse(bytes.NewReader(m))
		x.NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(w, r)

		res := w.Result()
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusOK, res.StatusCode, string(body))
		x.Equal(data, body)
		x.Equal("application/foo", res.Header.Get("Content-Type"))
		x.Equal(fmt.Sprintf("%d", len(data)), res.Header.Get("Content-Length"))
	})
	t.Run("412 if blob not found", func(t *testing.T) {
		x := require.New(t)
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer s.Close()

		repo, err := remote.NewRepository(fmt.Sprintf("%s/foo", s.URL[len("http://"):]))
		x.NoError(err)

		repo.Client = s.Client()
		repo.PlainHTTP = true
		h, ok := handler.Resolve(repo, oci.Descriptor{MediaType: oci.MediaTypeImageManifest}, "")
		x.True(ok)

		m, err := json.Marshal(oci.Manifest{
			Layers: []oci.Descriptor{{}},
		})
		x.NoError(err)

		err = h.Parse(bytes.NewReader(m))
		x.NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(w, r)

		res := w.Result()
		body, err := io.ReadAll(res.Body)
		x.NoError(err)
		x.Equal(http.StatusPreconditionFailed, res.StatusCode, string(body))
	})
	t.Run("412 if there are multiple layers", func(t *testing.T) {
		x := require.New(t)

		h, ok := handler.Resolve(nil, oci.Descriptor{MediaType: oci.MediaTypeImageManifest}, "")
		x.True(ok)

		m, err := json.Marshal(oci.Manifest{
			Layers: []oci.Descriptor{{}, {}},
		})
		x.NoError(err)

		err = h.Parse(bytes.NewReader(m))
		x.NoError(err)

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		h.ServeHTTP(w, r)

		res := w.Result()
		x.Equal(http.StatusPreconditionFailed, res.StatusCode)
	})
}
