package config

import (
	"errors"

	"github.com/lesomnus/oras-get/addr"
	"github.com/lesomnus/z"
)

type ServerConfig struct {
	Addr addr.Http
}

func (c *ServerConfig) Evaluate() error {
	z.FallbackP(&c.Addr, "localhost:5001")
	return errors.Join(
		z.CatErr(".addr", c.Addr.Evaluate()),
	)
}
