package handler

import (
	"net/http"
	"slices"

	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

type indexHandler struct {
	handler
	platformSpecific
	parser[oci.Index]
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i := slices.IndexFunc(h.manifest.Manifests, func(v oci.Descriptor) bool {
		p := v.Platform
		if p == nil {
			return false
		}

		os, arch, variant := h.platform.Split()
		return p.OS == os && p.Architecture == arch && p.Variant == variant
	})
	if i < 0 {
		http.Error(w, "no manifest for the specified platform", http.StatusNotFound)
		return
	}

	desc := h.manifest.Manifests[i]
	rc, err := h.repo.Manifests().Fetch(r.Context(), desc)
	if err != nil {
		http.Error(w, "fetch manifest: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	h2, ok := Resolve(h.repo, desc, h.platform)
	if !ok {
		panic("unreachable: failed to resolve manifest handler")
	}
	if err := h2.Parse(rc); err != nil {
		ManifestParseFailed(w, err)
		return
	}

	h2.ServeHTTP(w, r)
}
