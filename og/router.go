package og

import (
	"net/http"
	"strings"

	"github.com/lesomnus/oras-get/match"
	"github.com/lesomnus/oras-get/refs"
)

type Router struct {
	Upstreams map[string]Server
	Matchers  []match.Matcher
}

func (s *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	p, _ = strings.CutPrefix(p, "/")

	ref, err := refs.Parse(p)
	if err != nil {
		http.Error(w, "invalid reference", http.StatusBadRequest)
		return
	}

	for _, m := range s.Matchers {
		name, ok := m.Match(ref)
		if !ok {
			continue
		}

		s, ok := s.Upstreams[name]
		if !ok {
			continue
		}

		var h http.Handler = &s
		if ref.Domain() == "" {
			h = http.StripPrefix("/"+ref.Domain(), h)
		}

		h.ServeHTTP(w, r)
		return
	}

	http.Error(w, "no matching upstream", http.StatusNotFound)
}
