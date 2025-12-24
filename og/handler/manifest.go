package handler

import (
	"fmt"
	"io"
	"net/http"

	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

type manifestHandler struct {
	handler
	portable
	parser[oci.Manifest]
}

func (h *manifestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch len(h.manifest.Layers) {
	case 0:
		http.Error(w, "manifest has no layers", http.StatusPreconditionFailed)
	case 1:
		break
	default:
		http.Error(w, "manifest has multiple layers, only single-layer manifests are supported", http.StatusPreconditionFailed)
		return
	}

	layer := h.manifest.Layers[0]
	rc, err := h.repo.Blobs().Fetch(r.Context(), layer)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch blob: %s", err), http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", layer.MediaType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", layer.Size))
	if _, err := io.Copy(w, rc); err != nil {
		http.Error(w, fmt.Sprintf("failed to write blob data: %s", err), http.StatusInternalServerError)
		return
	}
}
