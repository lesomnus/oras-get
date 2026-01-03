package upstream

import (
	"fmt"
	"net/http"

	"github.com/lesomnus/oras-get/refs"
	"github.com/opencontainers/go-digest"
	"oras.land/oras-go/v2/registry"
)

type Repository struct {
	Upstream  Upstream
	Reference refs.Ref
	registry.Repository
}

func (repo Repository) Redirect(w http.ResponseWriter, r *http.Request, digest digest.Digest) {
	url := fmt.Sprintf("%s://%s/v2/%s/blobs/%s", repo.Upstream.Scheme, repo.Reference.Domain(), repo.Reference.Repo(), digest.String())
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
