package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/lesomnus/oras-get/og/platform"
	"github.com/lesomnus/z"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
)

type Handler interface {
	http.Handler
	Portable() bool
	Parse(r io.Reader) error
}

func Resolve(repo *remote.Repository, desc oci.Descriptor, platform platform.Platform) (Handler, bool) {
	h_ := handler{repo, desc, platform}

	var h Handler
	switch desc.MediaType {
	case oci.MediaTypeImageManifest:
		h = &manifestHandler{handler: h_}
	case oci.MediaTypeImageIndex:
		h = &indexHandler{handler: h_}
	default:
		return nil, false
	}

	return h, true
}

type handler struct {
	repo     *remote.Repository
	desc     oci.Descriptor
	platform platform.Platform
}

type portable struct{}

func (h portable) Portable() bool {
	return true
}

type platformSpecific struct{}

func (h platformSpecific) Portable() bool {
	return false
}

type parser[T any] struct {
	manifest *T
}

func (h *parser[T]) Parse(r io.Reader) error {
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return z.Err(err, "decode manifest")
	}

	h.manifest = &v
	return nil
}
