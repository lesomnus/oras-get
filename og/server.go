package og

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/distribution/reference"
	"github.com/lesomnus/oras-get/og/handler"
	"github.com/lesomnus/oras-get/og/platform"
	"oras.land/oras-go/v2/registry/remote"
)

type Server struct {
	// docker.io, ghcr.io, etc.
	domain string
	client *http.Client
}

func NewServer(domain string) *Server {
	return &Server{
		domain: domain,
		client: http.DefaultClient,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		break
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// r.URL.Path is like /<path>:<tag>[/<platform>]
	path, platform, err := parsePath(r.URL.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid request path: %s", err), http.StatusBadRequest)
		return
	}
	if platform != "" && platform.Arch() == "" {
		http.Error(w, "invalid platform string: no arch", http.StatusBadRequest)
		return
	}

	ref, err := reference.ParseNormalizedNamed(s.domain + path)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid reference: %s", err), http.StatusBadRequest)
		return
	}

	tag := ""
	if tagged, ok := ref.(reference.Tagged); !ok {
		http.Error(w, "reference is not tagged", http.StatusBadRequest)
		return
	} else {
		tag = tagged.Tag()
	}

	ctx := r.Context()
	repo, err := remote.NewRepository(ref.Name())
	if err != nil {
		s.fail(ctx, w, err)
		return
	}

	repo.PlainHTTP = true
	repo.Client = s.client
	desc, rc, err := repo.Manifests().FetchReference(ctx, tag)
	if err != nil {
		s.fail(ctx, w, err)
		return
	}
	defer rc.Close()

	h, ok := handler.Resolve(repo, desc, platform)
	if !ok {
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	if h.Portable() && platform != "" {
		http.Error(w, "target does not hold platform information", http.StatusBadRequest)
		return
	} else if !h.Portable() && platform == "" {
		http.Error(w, "target requires platform information", http.StatusBadRequest)
		return
	}
	if err := h.Parse(rc); err != nil {
		handler.ManifestParseFailed(w, err)
		return
	}

	h.ServeHTTP(w, r)
}

func (s *Server) fail(ctx context.Context, w http.ResponseWriter, err error) {
	fmt.Printf("err.Error(): %v\n", err.Error())
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func parsePath(s string) (string, platform.Platform, error) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", "", fmt.Errorf("invalid request path: tag not found")
	}

	j := strings.Index(s[i+1:], "/")
	if j < 0 {
		return s, "", nil
	} else {
		return s[:i+1+j], platform.Platform(s[i+1+j+1:]), nil
	}
}
