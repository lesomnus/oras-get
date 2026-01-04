package refs_test

import (
	"testing"

	"github.com/lesomnus/oras-get/refs"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tcs := []struct {
		input    string
		domain   string
		repo     string
		tag      string
		platform refs.Platform
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
			input:    "repo:tag/linux/amd64",
			repo:     "repo",
			tag:      "tag",
			platform: "linux/amd64",
		},
		{
			input: "path/to/repo:tag",
			repo:  "path/to/repo",
			tag:   "tag",
		},
		{
			input:    "path/to/repo:tag/linux/amd64",
			repo:     "path/to/repo",
			tag:      "tag",
			platform: "linux/amd64",
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
			input:    "example.com/repo:tag/linux/amd64",
			domain:   "example.com",
			repo:     "repo",
			tag:      "tag",
			platform: "linux/amd64",
		},
		{
			input:  "127.0.0.1/repo:tag",
			domain: "127.0.0.1",
			repo:   "repo",
			tag:    "tag",
		},
		{
			input:    "127.0.0.1/repo:tag/linux/amd64",
			domain:   "127.0.0.1",
			repo:     "repo",
			tag:      "tag",
			platform: "linux/amd64",
		},
		{
			input:    "127.0.0.1/path/to/repo:tag/linux/amd64",
			domain:   "127.0.0.1",
			repo:     "path/to/repo",
			tag:      "tag",
			platform: "linux/amd64",
		},
		{
			input:  "127.0.0.1:80/repo:tag",
			domain: "127.0.0.1:80",
			repo:   "repo",
			tag:    "tag",
		},
		{
			input:    "127.0.0.1:80/repo:tag/linux/amd64",
			domain:   "127.0.0.1:80",
			repo:     "repo",
			tag:      "tag",
			platform: "linux/amd64",
		},
		{
			input:    "127.0.0.1:80/path/to/repo:tag/linux/amd64",
			domain:   "127.0.0.1:80",
			repo:     "path/to/repo",
			tag:      "tag",
			platform: "linux/amd64",
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
			x.Equal(tc.platform, ref.Platform())
		})
	}
}
