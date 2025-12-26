package addr_test

import (
	"fmt"
	"testing"

	"github.com/lesomnus/oras-get/addr"
	"github.com/stretchr/testify/require"
)

func TestHttpAddr(t *testing.T) {
	t.Run("Split", func(t *testing.T) {
		tests := [][]string{
			{"", "", "", ""},
			{":", "", "", ""},
			{":80", "", "", "80"},
			{"localhost", "", "localhost", ""},
			{"localhost:", "", "localhost", ""},
			{"localhost:80", "", "localhost", "80"},
			{"localhost:icmp", "", "localhost", "icmp"},
			{"://localhost", "", "localhost", ""},
			{"://localhost:", "", "localhost", ""},
			{"://localhost:80", "", "localhost", "80"},
			{"http://localhost", "http", "localhost", ""},
			{"http://localhost:", "http", "localhost", ""},
			{"http://localhost:80", "http", "localhost", "80"},
		}
		for _, test := range tests {
			p := addr.Http(test[0])
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
			{":", ""},
			{":80", ":80"},
			{"localhost", "localhost"},
			{"localhost:", "localhost"},
			{"localhost:80", "localhost:80"},
			{"://localhost", "localhost"},
			{"://localhost:", "localhost"},
			{"://localhost:80", "localhost:80"},
			{"http://localhost", "http://localhost"},
			{"http://localhost:", "http://localhost"},
			{"http://localhost:80", "http://localhost:80"},
		}
		for _, test := range tests {
			t.Run(fmt.Sprintf("(%s)->%s", test[0], test[1]), func(t *testing.T) {
				x := require.New(t)
				p := addr.Http(test[0]).Normalized()
				x.Equal(addr.Http(test[1]), p)
			})
		}
	})
}
