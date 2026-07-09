// Package registrytest provides a minimal, in-memory OCI registry that speaks
// enough of the distribution spec's read API to drive oras-get's real oras-go
// client in tests, with no external registry or oras CLI required.
package registrytest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

type storedBlob struct {
	mediaType string
	data      []byte
}

// Registry is an in-memory content-addressable store served over HTTP.
type Registry struct {
	*httptest.Server

	mu    sync.Mutex
	blobs map[digest.Digest]storedBlob
	tags  map[string]map[string]digest.Digest // repo -> tag -> digest
}

// New starts a registry and registers cleanup with t.
func New(t *testing.T) *Registry {
	t.Helper()
	reg := &Registry{
		blobs: map[digest.Digest]storedBlob{},
		tags:  map[string]map[string]digest.Digest{},
	}
	reg.Server = httptest.NewServer(reg.handler())
	t.Cleanup(reg.Server.Close)
	return reg
}

// Host returns the registry host (the server URL without its scheme), suitable
// for remote.NewRegistry.
func (reg *Registry) Host() string {
	return strings.TrimPrefix(reg.Server.URL, "http://")
}

func (reg *Registry) put(mediaType string, data []byte) oci.Descriptor {
	d := digest.FromBytes(data)
	reg.mu.Lock()
	reg.blobs[d] = storedBlob{mediaType: mediaType, data: data}
	reg.mu.Unlock()
	return oci.Descriptor{MediaType: mediaType, Digest: d, Size: int64(len(data))}
}

// AddBlob stores data as a layer blob titled with the given file name and
// returns its descriptor (including the title annotation).
func (reg *Registry) AddBlob(mediaType, title string, data []byte) oci.Descriptor {
	desc := reg.put(mediaType, data)
	desc.Annotations = map[string]string{oci.AnnotationTitle: title}
	return desc
}

// AddManifest builds an image manifest from the given layers, stores it and
// returns its descriptor. If platform is non-nil it is attached to the returned
// descriptor so it can be used as an image index entry.
func (reg *Registry) AddManifest(layers []oci.Descriptor, platform *oci.Platform) oci.Descriptor {
	m := oci.Manifest{
		Versioned: specVersioned,
		MediaType: oci.MediaTypeImageManifest,
		Config:    oci.DescriptorEmptyJSON,
		Layers:    layers,
	}
	data, _ := json.Marshal(m)
	desc := reg.put(oci.MediaTypeImageManifest, data)
	desc.Platform = platform
	return desc
}

// AddIndex builds an image index from the given manifest descriptors, stores it
// and returns its descriptor.
func (reg *Registry) AddIndex(manifests []oci.Descriptor) oci.Descriptor {
	idx := oci.Index{
		Versioned: specVersioned,
		MediaType: oci.MediaTypeImageIndex,
		Manifests: manifests,
	}
	data, _ := json.Marshal(idx)
	return reg.put(oci.MediaTypeImageIndex, data)
}

// Tag associates repo:tag with the given descriptor.
func (reg *Registry) Tag(repo, tag string, desc oci.Descriptor) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if reg.tags[repo] == nil {
		reg.tags[repo] = map[string]digest.Digest{}
	}
	reg.tags[repo][tag] = desc.Digest
}

func (reg *Registry) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rest, ok := strings.CutPrefix(r.URL.Path, "/v2/")
		if !ok {
			if r.URL.Path == "/v2" {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
			return
		}
		if rest == "" { // GET /v2/
			w.WriteHeader(http.StatusOK)
			return
		}

		if repo, ok := strings.CutSuffix(rest, "/tags/list"); ok {
			reg.serveTags(w, repo)
			return
		}
		if repo, ref, ok := cut(rest, "/manifests/"); ok {
			reg.serveManifest(w, r, repo, ref)
			return
		}
		if _, ref, ok := cut(rest, "/blobs/"); ok {
			reg.serveContent(w, r, digest.Digest(ref))
			return
		}
		http.NotFound(w, r)
	})
}

func (reg *Registry) serveManifest(w http.ResponseWriter, r *http.Request, repo, ref string) {
	d, err := digest.Parse(ref)
	if err != nil {
		// Not a digest; resolve it as a tag.
		reg.mu.Lock()
		d = reg.tags[repo][ref]
		reg.mu.Unlock()
		if d == "" {
			http.Error(w, "manifest unknown", http.StatusNotFound)
			return
		}
	}
	reg.serveContent(w, r, d)
}

func (reg *Registry) serveContent(w http.ResponseWriter, r *http.Request, d digest.Digest) {
	reg.mu.Lock()
	b, ok := reg.blobs[d]
	reg.mu.Unlock()
	if !ok {
		http.Error(w, "blob unknown", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", b.mediaType)
	w.Header().Set("Docker-Content-Digest", d.String())
	w.Header().Set("Content-Length", strconv.Itoa(len(b.data)))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Write(b.data)
}

func (reg *Registry) serveTags(w http.ResponseWriter, repo string) {
	reg.mu.Lock()
	tags := make([]string, 0, len(reg.tags[repo]))
	for t := range reg.tags[repo] {
		tags = append(tags, t)
	}
	reg.mu.Unlock()
	sort.Strings(tags)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{Name: repo, Tags: tags})
}

func cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

var specVersioned = specs.Versioned{SchemaVersion: 2}
