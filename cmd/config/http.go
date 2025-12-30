package config

import (
	"errors"

	"github.com/lesomnus/oras-get/addr"
	"github.com/lesomnus/z"
)

type HttpConfig struct {
	Addr addr.Http
	// TLS?
}

func (c *HttpConfig) Evaluate() error {
	return errors.Join(
		z.CatErr(".addr", c.Addr.Evaluate()),
	)
}
