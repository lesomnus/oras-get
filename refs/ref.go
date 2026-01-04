package refs

import (
	"strings"

	"github.com/distribution/reference"
	"github.com/lesomnus/z"
)

type Ref string

func Parse(s string) (Ref, error) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return Ref(s), nil
	}

	p := ""
	j := strings.Index(s[i+1:], "/")
	if j >= 0 {
		p = s[i+1+j+1:]
		s = s[:i+1+j]
	}

	v, err := reference.Parse(s)
	if err != nil {
		return "", err
	}

	w, ok := v.(reference.NamedTagged)
	if !ok {
		return "", z.Err(reference.ErrReferenceInvalidFormat, "reference must be tagged")
	}

	return buildRef(reference.Domain(w), reference.Path(w), w.Tag(), Platform(p)), nil
}

func buildRef(domain, repo, tag string, platform Platform) Ref {
	r := ""
	if domain != "" {
		r += domain + "/"
	}
	r += repo + ":" + tag
	if platform != "" {
		r += "/" + string(platform)
	}
	return Ref(r)
}

func (r Ref) Split() (domain string, repo string, tag string, platform Platform) {
	// [domain/]<repo>:<tag>[/<platform>]
	if i := strings.LastIndex(string(r), ":"); i < 0 {
		repo = string(r)
	} else {
		repo = string(r[:i])
		tag = string(r[i+1:])
	}

	// Find domain
	if i := strings.Index(repo, "/"); i >= 0 {
		d := repo[:i]
		if strings.Contains(d, ".") || strings.Contains(d, ":") {
			domain = d
			repo = repo[i+1:]
		}
	}

	// Find platform
	if i := strings.Index(tag, "/"); i >= 0 {
		platform = Platform(tag[i+1:])
		tag = tag[:i]
	}

	return
}

func (r Ref) Domain() string {
	v, _, _, _ := r.Split()
	return v
}

func (r Ref) Repo() string {
	_, v, _, _ := r.Split()
	return v
}

func (r Ref) Tag() string {
	_, _, v, _ := r.Split()
	return v
}

func (r Ref) Platform() Platform {
	_, _, _, v := r.Split()
	return v
}

func (r Ref) Name() string {
	domain, repo, _, _ := r.Split()
	if domain != "" {
		return domain + "/" + repo
	}
	return repo
}

func WithDomain(r Ref, d string) Ref {
	_, repo, tag, p := r.Split()
	return buildRef(d, repo, tag, p)
}

func WithPlatform(r Ref, p Platform) Ref {
	domain, repo, tag, _ := r.Split()
	return buildRef(domain, repo, tag, p)
}
