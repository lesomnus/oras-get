package handler

import (
	"errors"
	"fmt"
	"net/http"
	"slices"

	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
)

type indexHandler struct {
	handler
	platformSpecific
	parser[oci.Index]
}

func (h *indexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := h.Repo.Reference.Platform().Normalized()
	f := func(v oci.Descriptor) bool {
		vp := v.Platform
		if vp == nil {
			return false
		}

		os, arch, variant := p.Split()
		return vp.OS == os && vp.Architecture == arch && vp.Variant == variant
	}

	// Find normalized platform first, then try original.
	i := slices.IndexFunc(h.manifest.Manifests, f)
	if i < 0 {
		p = h.Repo.Reference.Platform()
		i = slices.IndexFunc(h.manifest.Manifests, f)
	}
	if i < 0 {
		http.Error(w, "no manifest for the specified platform", http.StatusNotFound)
		return
	}

	desc := h.manifest.Manifests[i]
	rc, err := h.Repo.Manifests().Fetch(r.Context(), desc)
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusPreconditionFailed)
		} else {
			http.Error(w, fmt.Sprintf("fetch manifest: %s", err), http.StatusInternalServerError)
		}
		return
	}
	defer rc.Close()

	h2, ok := Resolve(h.Repo, desc)
	if !ok {
		panic("unreachable: failed to resolve manifest handler")
	}
	if err := h2.Parse(rc); err != nil {
		ManifestParseFailed(w, err)
		return
	}

	h2.ServeHTTP(w, r)
}
