package match

import (
	"strings"
	"text/template"

	"github.com/lesomnus/oras-get/refs"
)

type Matcher interface {
	Match(ref refs.Ref) (string, bool)
}

type FixedMatcher string

func (m FixedMatcher) Match(ref refs.Ref) (string, bool) {
	return string(m), true
}

type StaticNamedMatcher struct {
	Domain *string
	Repo   *string

	Next string
}

func (m *StaticNamedMatcher) Match(ref refs.Ref) (_ string, _ bool) {
	if m.Domain != nil {
		domain := ref.Domain()
		if domain != *m.Domain {
			return
		}
	}
	if m.Repo != nil {
		repo := ref.Repo()
		if repo != *m.Repo {
			return
		}
	}
	return m.Next, true
}

type TemplateMatcher struct {
	Next *template.Template
}

type TemplateMatcherValue struct {
	Domain string
	Repo   string
}

func (m *TemplateMatcher) Match(ref refs.Ref) (_ string, _ bool) {
	v := TemplateMatcherValue{
		Domain: ref.Domain(),
		Repo:   ref.Repo(),
	}

	var next strings.Builder
	if err := m.Next.Execute(&next, &v); err != nil {
		return
	}

	return next.String(), true
}
