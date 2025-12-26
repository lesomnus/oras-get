package refs_test

import (
	"fmt"
	"testing"

	"github.com/lesomnus/oras-get/refs"
	"github.com/stretchr/testify/require"
)

func TestPlatform(t *testing.T) {
	t.Run("Split", func(t *testing.T) {
		tests := [][]string{
			{"", "", "", ""},
			{"/", "", "", ""},
			{"//", "", "", ""},
			{"///", "", "", "/"},
			{"////", "", "", "//"},
			{"linux", "linux", "", ""},
			{"linux/amd64", "linux", "amd64", ""},
			{"linux/amd64/v8", "linux", "amd64", "v8"},
			{"linux/", "linux", "", ""},
			{"linux//", "linux", "", ""},
			{"linux//v7", "linux", "", "v7"},
			{"linux//v7/", "linux", "", "v7/"},
		}
		for _, test := range tests {
			p := refs.Platform(test[0])
			a, b, c := p.Split()
			t.Run(fmt.Sprintf("(%s)->%s,%s,%s", p, a, b, c), func(t *testing.T) {
				x := require.New(t)
				x.Equal(test[1], a)
				x.Equal(test[2], b)
				x.Equal(test[3], c)
			})
		}
	})
	t.Run("Normalized", func(t *testing.T) {
		tests := [][]string{
			{"", ""},
			{"/", ""},
			{"//", ""},
			{"///", ""},
			{"////", ""},
			{"linux", "linux"},
			{"linux/", "linux"},
			{"linux//", "linux"},
			{"linux///", "linux"},
			{"linux////", "linux"},
			{"linux/arm64", "linux/arm64"},
			{"linux/arm64/", "linux/arm64"},
			{"linux/arm64//", "linux/arm64"},
			{"linux/arm64///", "linux/arm64"},
			{"linux/arm64/v8", "linux/arm64/v8"},
			{"linux/arm64/v8/", "linux/arm64/v8"},
			{"linux/arm64/v8//", "linux/arm64/v8"},
			{"linux//v8", "linux"},
			{"linux//v8/", "linux"},
			{"linux//v8//", "linux"},
			{"linux//v8", "linux"},

			{"linux/aarch64", "linux/arm64"},
			{"linux/aarch64/", "linux/arm64"},
			{"linux/aarch64//", "linux/arm64"},
			{"linux/aarch64///", "linux/arm64"},
			{"linux/aarch64/v8", "linux/arm64/v8"},
			{"linux/aarch64/v8/", "linux/arm64/v8"},
			{"linux/aarch64/v8//", "linux/arm64/v8"},

			{"linux/x86_64", "linux/amd64"},
			{"linux/aarch32", "linux/arm"},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("(%s)->%s", test[0], test[1]), func(t *testing.T) {
				x := require.New(t)
				p := refs.Platform(test[0]).Normalized()
				x.Equal(refs.Platform(test[1]), p)
			})
		}
	})
}
