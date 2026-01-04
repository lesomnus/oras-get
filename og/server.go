package og

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/lesomnus/oras-get/og/handler"
	"github.com/lesomnus/oras-get/og/upstream"
	"github.com/lesomnus/oras-get/refs"
	"github.com/lesomnus/otx/log"
	"oras.land/oras-go/v2/errdef"
)

type server struct {
	Upstream  upstream.Upstream
	Reference refs.Ref
}

func (s server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ref := s.Reference
	repo, err := s.Upstream.Repository(ctx, ref)
	if err != nil {
		s.fail(ctx, w, err)
		return
	}
	if ref.Tag() == "_" {
		serveTagList(w, r, repo)
		return
	}

	desc, rc, err := repo.Manifests().FetchReference(ctx, ref.Tag())
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		s.fail(ctx, w, err)
		return
	}
	defer rc.Close()

	h, ok := handler.Resolve(repo, desc)
	if !ok {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	if h.Portable() && ref.Platform() != "" {
		http.Error(w, "target does not hold platform information", http.StatusBadRequest)
		return
	} else if !h.Portable() && ref.Platform() == "" {
		http.Error(w, "target requires platform information", http.StatusBadRequest)
		return
	}
	if err := h.Parse(rc); err != nil {
		handler.ManifestParseFailed(w, err)
		return
	}

	h.ServeHTTP(w, r)
}

func (s server) fail(ctx context.Context, w http.ResponseWriter, err error) {
	log.From(ctx).Warn("request failed", slog.String("err", err.Error()))
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
