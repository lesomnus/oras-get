package handler

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/opencontainers/go-digest"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
)

type manifestHandler struct {
	handler
	portable
	parser[oci.Manifest]
}

func (h *manifestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	layers := h.manifest.Layers
	ref := h.Repo.Reference

	if ref.ListFiles() {
		serveFileList(w, r, layers, h.Desc.Digest)
		return
	}

	layer, code, msg := selectLayer(layers, ref.File())
	if code != 0 {
		http.Error(w, msg, code)
		return
	}

	// A layer's digest is a strong, content-addressed validator. Expose it as an
	// ETag so clients (browsers, CDNs, proxies) can cache the file and revalidate
	// cheaply. Because a tag may be re-pushed, we ask clients to revalidate
	// rather than assume freshness; when they do, an unchanged file costs only a
	// 304 and never re-fetches the blob from upstream.
	etag := etagOf(layer.Digest)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	if ifNoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if h.Repo.Upstream.Redirect {
		h.Repo.Redirect(w, r, layer.Digest)
		return
	}
	w.Header().Set("Content-Type", layer.MediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", layer.Size))
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	rc, err := h.Repo.Blobs().Fetch(r.Context(), layer)
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
		} else {
			http.Error(w, fmt.Sprintf("fetch blob: %s", err), http.StatusInternalServerError)
		}
		return
	}
	defer rc.Close()

	if _, err := io.Copy(w, rc); err != nil {
		http.Error(w, fmt.Sprintf("write blob data: %s", err), http.StatusInternalServerError)
		return
	}
}

// selectLayer picks the layer to serve from an artifact's layers given the
// requested file path. A zero code means success; otherwise it returns the HTTP
// status and message to respond with.
//
// When no file is requested, a single-layer artifact is served as-is (so simple
// single-file artifacts need no path), while a multi-file artifact is a bad
// request since the caller must disambiguate which file it wants.
func selectLayer(layers []oci.Descriptor, file string) (_ oci.Descriptor, code int, msg string) {
	if file == "" {
		switch len(layers) {
		case 0:
			return oci.Descriptor{}, http.StatusPreconditionFailed, "manifest has no layers"
		case 1:
			return layers[0], 0, ""
		default:
			return oci.Descriptor{}, http.StatusBadRequest,
				"artifact holds multiple files; append the file path to the reference. available files:\n" +
					strings.Join(fileNames(layers), "\n")
		}
	}

	// Prefer an exact title match so that distinctly-titled layers are each
	// addressable by their own name.
	for _, l := range layers {
		if fileTitle(l) == file {
			return l, 0, ""
		}
	}
	// Fall back to a normalized comparison so a caller may include a leading
	// "./" or "/" or redundant separators.
	want := normFile(file)
	for _, l := range layers {
		if t := fileTitle(l); t != "" && normFile(t) == want {
			return l, 0, ""
		}
	}
	return oci.Descriptor{}, http.StatusNotFound, fmt.Sprintf("no file %q in artifact", file)
}

func serveFileList(w http.ResponseWriter, r *http.Request, layers []oci.Descriptor, manifest digest.Digest) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// The listing is fully determined by the manifest, so its digest is a valid
	// ETag for conditional requests.
	if manifest != "" {
		etag := etagOf(manifest)
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")
		if ifNoneMatch(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	bw := bufio.NewWriter(w)
	defer bw.Flush()
	for _, name := range fileNames(layers) {
		fmt.Fprintln(bw, name)
	}
}

// etagOf renders a strong ETag from a content digest.
func etagOf(d digest.Digest) string {
	return `"` + d.String() + `"`
}

// ifNoneMatch reports whether the request's If-None-Match precondition matches
// etag, i.e. the client already holds the current content (RFC 9110 §13.1.2).
func ifNoneMatch(r *http.Request, etag string) bool {
	h := r.Header.Get("If-None-Match")
	if h == "" {
		return false
	}
	if strings.TrimSpace(h) == "*" {
		return true
	}
	for _, part := range strings.Split(h, ",") {
		// If-None-Match uses the weak comparison, so a "W/" prefix is ignored.
		if strings.TrimPrefix(strings.TrimSpace(part), "W/") == strings.TrimPrefix(etag, "W/") {
			return true
		}
	}
	return false
}

// fileNames returns the addressable file names in manifest order. Layers
// without a title annotation are omitted: they are not addressable by name, so
// advertising them would only list something a caller cannot fetch.
func fileNames(layers []oci.Descriptor) []string {
	names := make([]string, 0, len(layers))
	for _, l := range layers {
		if t := fileTitle(l); t != "" {
			names = append(names, t)
		}
	}
	return names
}

// fileTitle returns the file name of a layer as recorded in its
// "org.opencontainers.image.title" annotation, or "" if it has none.
func fileTitle(d oci.Descriptor) string {
	return d.Annotations[oci.AnnotationTitle]
}

// normFile normalizes a file path for comparison. It collapses "." and
// redundant separators and strips any leading "/" or "./", so that "bin/tool",
// "./bin/tool" and "/bin/tool" all compare equal.
func normFile(s string) string {
	if s == "" {
		return ""
	}
	return path.Clean("/" + s)
}
