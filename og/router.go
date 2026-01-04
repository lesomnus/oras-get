package og

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/lesomnus/oras-get/match"
	"github.com/lesomnus/oras-get/og/upstream"
	"github.com/lesomnus/oras-get/refs"
	"github.com/lesomnus/otx/log"
)

type Router struct {
	Upstreams map[string]upstream.Upstream
	Matchers  []match.Matcher
}

func (s *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		break
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p := r.URL.Path
	p, _ = strings.CutPrefix(p, "/")

	ref, err := refs.Parse(p)
	if err != nil {
		http.Error(w, "invalid reference", http.StatusBadRequest)
		return
	}
	if ref.Tag() == "" {
		http.Error(w, "reference must be tagged", http.StatusBadRequest)
		return
	}
	if ref.Platform() != "" && ref.Platform().Arch() == "" {
		http.Error(w, "invalid platform string: no arch", http.StatusBadRequest)
		return
	}

	for i, m := range s.Matchers {
		name, ok := m.Match(ref)
		if !ok {
			continue
		}

		upstream, ok := s.Upstreams[name]
		if !ok {
			continue
		}

		ctx := r.Context()
		l := log.From(ctx)
		l = l.With(slog.String("upstream", name))
		l.Info("route",
			slog.Int("matcher", i),
			slog.String("ref", ref.Name()),
		)

		ctx = log.Into(ctx, l)
		r = r.WithContext(ctx)
		server{upstream, ref}.serveHTTP(w, r)
		return
	}

	http.Error(w, "no matching upstream", http.StatusNotFound)
}
