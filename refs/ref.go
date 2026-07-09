package refs

import (
	"strings"

	"github.com/distribution/reference"
)

// Ref is an artifact reference, kept as the raw string and re-parsed on demand
// so that the string is always the single source of truth. Its form mirrors the
// request URL:
//
//	[<domain>/]<repo>:<tag>[/<file>][?platform=<platform>]
//
// The file component selects a single file (an OCI layer, identified by its
// "org.opencontainers.image.title" annotation) within the target manifest. A
// trailing slash with an empty file (e.g. "repo:tag/") requests a listing of
// the available files instead.
//
// The platform is not part of the path; it selects a manifest out of an image
// index and is carried as a "?platform=" suffix, populated from the request's
// query parameter via WithPlatform.
type Ref string

const platformMark = "?platform="

// Parse validates s (which must be in path form, without a platform query) and
// returns it as a Ref. The reference grammar is checked against the
// domain/repo/tag portion only; the file path may legally contain characters
// (e.g. colons) that a bare reference may not.
func Parse(s string) (Ref, error) {
	domain, repo, tag, file, list, platform := Ref(s).Split()

	name := repo
	if domain != "" {
		name = domain + "/" + repo
	}
	if tag != "" {
		name += ":" + tag
	}
	if _, err := reference.Parse(name); err != nil {
		return "", err
	}

	return build(domain, repo, tag, file, list, platform), nil
}

// Split decomposes the reference every time it is called. Nothing is cached, so
// there is no way for the parts to drift out of sync with the string.
func (r Ref) Split() (domain, repo, tag, file string, list bool, platform Platform) {
	s := string(r)

	// Peel off the platform query, if any.
	if i := strings.Index(s, platformMark); i >= 0 {
		platform = Platform(s[i+len(platformMark):])
		s = s[:i]
	}

	// Peel off an optional leading domain. This must happen before we look for
	// the tag separator: a domain may carry a port colon and the file path may
	// carry colons of its own, so scanning for a ":" over the whole string would
	// pick the wrong one. The first segment is a domain only when it looks like
	// a host (a "." or a "host:port"), matching how a request URL without an
	// explicit domain is read; otherwise it is part of the repo.
	rest := s
	if i := strings.Index(s, "/"); i >= 0 && looksLikeDomain(s[:i]) {
		domain = s[:i]
		rest = s[i+1:]
	}

	// rest is now "repo:tag[/file]". A repository path never contains a colon,
	// so the first colon is the tag separator, and the file (if any) begins at
	// the first slash following the tag.
	repo = rest
	if i := strings.Index(rest, ":"); i >= 0 {
		repo = rest[:i]
		tag = rest[i+1:]
		if j := strings.Index(tag, "/"); j >= 0 {
			file = tag[j+1:]
			list = file == ""
			tag = tag[:j]
		}
	}
	return
}

// build renders the canonical reference string from its parts. It is the
// inverse of Split.
func build(domain, repo, tag, file string, list bool, platform Platform) Ref {
	var b strings.Builder
	if domain != "" {
		b.WriteString(domain)
		b.WriteByte('/')
	}
	b.WriteString(repo)
	if tag != "" {
		b.WriteByte(':')
		b.WriteString(tag)
	}
	if file != "" {
		b.WriteByte('/')
		b.WriteString(file)
	} else if list {
		b.WriteByte('/')
	}
	if platform != "" {
		b.WriteString(platformMark)
		b.WriteString(string(platform))
	}
	return Ref(b.String())
}

// looksLikeDomain reports whether a leading path segment should be read as a
// registry host rather than as the first component of a repository path.
func looksLikeDomain(s string) bool {
	if strings.Contains(s, ".") {
		return true
	}
	// A "host:port" form, where the port is numeric.
	if i := strings.LastIndex(s, ":"); i >= 0 {
		port := s[i+1:]
		if port == "" {
			return false
		}
		for _, r := range port {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}
	return false
}

func (r Ref) Domain() string {
	v, _, _, _, _, _ := r.Split()
	return v
}

func (r Ref) Repo() string {
	_, v, _, _, _, _ := r.Split()
	return v
}

func (r Ref) Tag() string {
	_, _, v, _, _, _ := r.Split()
	return v
}

// File returns the selected file path within the target artifact, or "" if no
// specific file was requested.
func (r Ref) File() string {
	_, _, _, v, _, _ := r.Split()
	return v
}

// ListFiles reports whether a listing of the available files was requested
// (i.e. the reference ended with a trailing slash, "repo:tag/").
func (r Ref) ListFiles() bool {
	_, _, _, _, v, _ := r.Split()
	return v
}

func (r Ref) Platform() Platform {
	_, _, _, _, _, v := r.Split()
	return v
}

func (r Ref) Name() string {
	domain, repo, _, _, _, _ := r.Split()
	if domain != "" {
		return domain + "/" + repo
	}
	return repo
}

func WithDomain(r Ref, d string) Ref {
	_, repo, tag, file, list, platform := r.Split()
	return build(d, repo, tag, file, list, platform)
}

func WithPlatform(r Ref, p Platform) Ref {
	domain, repo, tag, file, list, _ := r.Split()
	return build(domain, repo, tag, file, list, p)
}

func WithFile(r Ref, f string) Ref {
	domain, repo, tag, _, _, platform := r.Split()
	return build(domain, repo, tag, f, false, platform)
}
