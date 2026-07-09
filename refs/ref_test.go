package refs_test

import (
	"testing"

	"github.com/lesomnus/oras-get/refs"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tcs := []struct {
		input  string
		domain string
		repo   string
		tag    string
		file   string
		list   bool
	}{
		{
			input: "repo",
			repo:  "repo",
		},
		{
			input: "path/to/repo",
			repo:  "path/to/repo",
		},
		{
			input: "repo:tag",
			repo:  "repo",
			tag:   "tag",
		},
		{
			input: "repo:v1.0.0",
			repo:  "repo",
			tag:   "v1.0.0",
		},
		{
			input: "repo:tag/bin/tool",
			repo:  "repo",
			tag:   "tag",
			file:  "bin/tool",
		},
		{
			input: "repo:tag/README.md",
			repo:  "repo",
			tag:   "tag",
			file:  "README.md",
		},
		{
			// Trailing slash requests a file listing.
			input: "repo:tag/",
			repo:  "repo",
			tag:   "tag",
			list:  true,
		},
		{
			input: "path/to/repo:tag",
			repo:  "path/to/repo",
			tag:   "tag",
		},
		{
			input: "path/to/repo:tag/bin/tool",
			repo:  "path/to/repo",
			tag:   "tag",
			file:  "bin/tool",
		},
		{
			input:  "example.com/repo:tag",
			domain: "example.com",
			repo:   "repo",
			tag:    "tag",
		},
		{
			input:  "example.com/path/to/repo:tag",
			domain: "example.com",
			repo:   "path/to/repo",
			tag:    "tag",
		},
		{
			input:  "example.com/repo:tag/dir/file.bin",
			domain: "example.com",
			repo:   "repo",
			tag:    "tag",
			file:   "dir/file.bin",
		},
		{
			input:  "127.0.0.1/repo:tag",
			domain: "127.0.0.1",
			repo:   "repo",
			tag:    "tag",
		},
		{
			input:  "127.0.0.1/repo:tag/a",
			domain: "127.0.0.1",
			repo:   "repo",
			tag:    "tag",
			file:   "a",
		},
		{
			input:  "127.0.0.1/path/to/repo:tag/a/b/c",
			domain: "127.0.0.1",
			repo:   "path/to/repo",
			tag:    "tag",
			file:   "a/b/c",
		},
		{
			input:  "127.0.0.1:80/repo:tag",
			domain: "127.0.0.1:80",
			repo:   "repo",
			tag:    "tag",
		},
		{
			input:  "127.0.0.1:80/repo:tag/bin/tool",
			domain: "127.0.0.1:80",
			repo:   "repo",
			tag:    "tag",
			file:   "bin/tool",
		},
		{
			input:  "127.0.0.1:80/path/to/repo:tag/bin/tool",
			domain: "127.0.0.1:80",
			repo:   "path/to/repo",
			tag:    "tag",
			file:   "bin/tool",
		},
		{
			// A colon is legal in a file title and must not be mistaken for the
			// tag separator.
			input: "repo:tag/my:file.txt",
			repo:  "repo",
			tag:   "tag",
			file:  "my:file.txt",
		},
		{
			input: "repo:tag/dir/foo:bar.txt",
			repo:  "repo",
			tag:   "tag",
			file:  "dir/foo:bar.txt",
		},
		{
			input:  "example.com/repo:tag/weird:name",
			domain: "example.com",
			repo:   "repo",
			tag:    "tag",
			file:   "weird:name",
		},
		{
			// Port on the domain plus a colon in the file: both colons must land
			// in the right place.
			input:  "127.0.0.1:5000/repo:tag/a:b",
			domain: "127.0.0.1:5000",
			repo:   "repo",
			tag:    "tag",
			file:   "a:b",
		},
		{
			// A tagless reference whose domain has a port must not have the port
			// mistaken for a tag; the tag stays empty (the router rejects it).
			input:  "127.0.0.1:80/repo",
			domain: "127.0.0.1:80",
			repo:   "repo",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.input, func(t *testing.T) {
			x := require.New(t)

			ref, err := refs.Parse(tc.input)
			x.NoError(err)
			x.Equal(tc.domain, ref.Domain())
			x.Equal(tc.repo, ref.Repo())
			x.Equal(tc.tag, ref.Tag())
			x.Equal(tc.file, ref.File())
			x.Equal(tc.list, ref.ListFiles())
			// Platform is never derived from the path.
			x.Equal(refs.Platform(""), ref.Platform())
		})
	}
}

func TestString(t *testing.T) {
	// The Ref is the raw string; With* render the canonical URL-like form, and
	// re-parsing that string yields the same parts (the string is the single
	// source of truth).
	x := require.New(t)

	ref, err := refs.Parse("example.com/app:v1/bin/tool")
	x.NoError(err)
	x.Equal("example.com/app:v1/bin/tool", string(ref))

	p := refs.WithPlatform(ref, refs.Platform("linux/amd64"))
	x.Equal("example.com/app:v1/bin/tool?platform=linux/amd64", string(p))
	x.Equal(refs.Platform("linux/amd64"), p.Platform())
	x.Equal("bin/tool", p.File())
	x.Equal("app", p.Repo())
	x.Equal("example.com", p.Domain())
	x.Equal("v1", p.Tag())
}

func TestWith(t *testing.T) {
	x := require.New(t)

	ref, err := refs.Parse("example.com/repo:tag/bin/tool")
	x.NoError(err)

	t.Run("WithPlatform", func(t *testing.T) {
		r := refs.WithPlatform(ref, refs.Platform("linux/amd64"))
		x.Equal(refs.Platform("linux/amd64"), r.Platform())
		// Original is unchanged; other fields are preserved.
		x.Equal(refs.Platform(""), ref.Platform())
		x.Equal("bin/tool", r.File())
		x.Equal("example.com", r.Domain())
	})
	t.Run("WithDomain", func(t *testing.T) {
		r := refs.WithDomain(ref, "other.com")
		x.Equal("other.com", r.Domain())
		x.Equal("example.com", ref.Domain())
		x.Equal("bin/tool", r.File())
	})
	t.Run("WithFile", func(t *testing.T) {
		r := refs.WithFile(ref, "other/file")
		x.Equal("other/file", r.File())
		x.False(r.ListFiles())
		x.Equal("bin/tool", ref.File())
	})
}
