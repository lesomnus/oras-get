package match_test

import (
	"runtime"
	"testing"
	"text/template"

	"github.com/lesomnus/oras-get/match"
	"github.com/lesomnus/oras-get/refs"
	"github.com/lesomnus/z"
	"github.com/stretchr/testify/require"
)

func TestFixedMatcher(t *testing.T) {
	x := require.New(t)

	m := match.FixedMatcher("foo")
	for _, s := range []string{
		"repo:tag",
		"example.com/repo:tag",
	} {
		ref, err := refs.Parse(s)
		if err != nil {
			panic(err)
		}

		v, ok := m.Match(ref)
		x.True(ok)
		x.Equal("foo", v)
	}
}

func TestStaticNamedMatcher(t *testing.T) {
	o_ := "o"
	_x := "x"

	t.Run("domain", func(t *testing.T) {
		x := require.New(t)

		m := &match.StaticNamedMatcher{Domain: z.Ptr("example.com")}
		for _, tc := range [][]string{
			{_x, "repo:tag"},
			{_x, "path/repo:tag"},
			{_x, "path/a/repo:tag"},
			{_x, "path/b/repo:tag"},

			{_x, "127.0.0.1/repo:tag"},
			{_x, "127.0.0.1/path/repo:tag"},
			{_x, "127.0.0.1/path/a/repo:tag"},
			{_x, "127.0.0.1/path/b/repo:tag"},

			{_x, "127.0.0.1:80/repo:tag"},
			{_x, "127.0.0.1:80/path/repo:tag"},
			{_x, "127.0.0.1:80/path/a/repo:tag"},
			{_x, "127.0.0.1:80/path/b/repo:tag"},

			{o_, "example.com/repo:tag"},
			{o_, "example.com/path/repo:tag"},
			{o_, "example.com/path/a/repo:tag"},
			{o_, "example.com/path/b/repo:tag"},

			{_x, "example.com:80/repo:tag"},
			{_x, "example.com:80/path/repo:tag"},
			{_x, "example.com:80/path/a/repo:tag"},
			{_x, "example.com:80/path/b/repo:tag"},

			{_x, "other.com/repo:tag"},
			{_x, "other.com/path/repo:tag"},
			{_x, "other.com/path/a/repo:tag"},
			{_x, "other.com/path/b/repo:tag"},

			{_x, "other.com:80/repo:tag"},
			{_x, "other.com:80/path/repo:tag"},
			{_x, "other.com:80/path/a/repo:tag"},
			{_x, "other.com:80/path/b/repo:tag"},
		} {
			o := tc[0] == o_
			s := tc[1]
			if len(tc) == 3 {
				runtime.Breakpoint()
			}

			ref, err := refs.Parse(s)
			if err != nil {
				panic(z.Err(err, "%s", s))
			}

			_, ok := m.Match(ref)
			x.Equal(o, ok, s)
		}
	})
	t.Run("repo", func(t *testing.T) {
		x := require.New(t)

		m := &match.StaticNamedMatcher{Repo: z.Ptr("path/a/repo")}
		for _, tc := range [][]string{
			{_x, "repo:tag"},
			{_x, "path/repo:tag"},
			{o_, "path/a/repo:tag"},
			{_x, "path/b/repo:tag"},

			{_x, "127.0.0.1/repo:tag"},
			{_x, "127.0.0.1/path/repo:tag"},
			{o_, "127.0.0.1/path/a/repo:tag"},
			{_x, "127.0.0.1/path/b/repo:tag"},

			{_x, "127.0.0.1:80/repo:tag"},
			{_x, "127.0.0.1:80/path/repo:tag"},
			{o_, "127.0.0.1:80/path/a/repo:tag"},
			{_x, "127.0.0.1:80/path/b/repo:tag"},

			{_x, "example.com/repo:tag"},
			{_x, "example.com/path/repo:tag"},
			{o_, "example.com/path/a/repo:tag"},
			{_x, "example.com/path/b/repo:tag"},

			{_x, "example.com:80/repo:tag"},
			{_x, "example.com:80/path/repo:tag"},
			{o_, "example.com:80/path/a/repo:tag"},
			{_x, "example.com:80/path/b/repo:tag"},

			{_x, "other.com/repo:tag"},
			{_x, "other.com/path/repo:tag"},
			{o_, "other.com/path/a/repo:tag"},
			{_x, "other.com/path/b/repo:tag"},

			{_x, "other.com:80/repo:tag"},
			{_x, "other.com:80/path/repo:tag"},
			{o_, "other.com:80/path/a/repo:tag"},
			{_x, "other.com:80/path/b/repo:tag"},
		} {
			o := tc[0] == o_
			s := tc[1]
			if len(tc) == 3 {
				runtime.Breakpoint()
			}

			ref, err := refs.Parse(s)
			if err != nil {
				panic(z.Err(err, "%s", s))
			}

			_, ok := m.Match(ref)
			x.Equal(o, ok, s)
		}
	})
}

func TestTemplateMatcher(t *testing.T) {
	x := require.New(t)

	ref, err := refs.Parse("example.com:80/repo:tag")
	x.NoError(err)

	tmpl, err := template.New("").Parse("{{ .Domain }}/{{ .Repo }}")
	x.NoError(err)

	m := match.TemplateMatcher{Next: tmpl}
	next, ok := m.Match(ref)
	x.True(ok)
	x.Equal("example.com:80/repo", next)
}
