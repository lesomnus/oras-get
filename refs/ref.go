package refs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/distribution/reference"
	"github.com/lesomnus/z"
)

type Ref interface {
	Domain() string
	Repo() string
	Tag() string
	Platform() Platform
}

func Name(ref Ref) string {
	domain := ref.Domain()
	if domain == "" {
		return ref.Repo()
	}

	return fmt.Sprintf("%s/%s", domain, ref.Repo())
}

type ref struct {
	domain   string
	repo     string
	tag      string
	platform Platform
}

func (r *ref) String() string {
	s := ""
	if r.domain != "" {
		s += r.domain + "/"
	}
	s += r.repo + ":" + r.tag
	if p := r.platform.String(); p != "" {
		s += "/" + p
	}

	return s
}

func (r *ref) Domain() string {
	return r.domain
}

func (r *ref) Repo() string {
	return r.repo
}

func (r *ref) Tag() string {
	return r.tag
}

func (r *ref) Platform() Platform {
	return r.platform
}

func Parse(s string) (Ref, error) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return nil, errors.New("tag not found")
	}

	p := ""
	j := strings.Index(s[i+1:], "/")
	if j >= 0 {
		p = s[i+1+j+1:]
		s = s[:i+1+j]
	}

	v, err := reference.Parse(s)
	if err != nil {
		return nil, err
	}

	w, ok := v.(reference.NamedTagged)
	if !ok {
		return nil, z.Err(reference.ErrReferenceInvalidFormat, "reference must be tagged")
	}

	r := &ref{
		domain:   reference.Domain(w),
		repo:     reference.Path(w),
		tag:      w.Tag(),
		platform: Platform(p),
	}

	domain := reference.Domain(w)
	if domain == "" {
		return r, nil
	}
	if strings.ContainsAny(domain, ".:[]") {
		return r, nil
	}

	r.domain = ""
	r.repo = w.Name()
	return r, nil
}
