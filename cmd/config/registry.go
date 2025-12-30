package config

import (
	"errors"
	"net/http"

	"github.com/lesomnus/oras-get/addr"
	"github.com/lesomnus/otx/otxhttp"
	"github.com/lesomnus/z"
	"oras.land/oras-go/v2/registry/remote"
)

type RegistryConfig struct {
	Addr addr.Http
	// TLS Verify?
	// Prefix?
	// Auth?
	// Forwarded Headers?
}

func (u *RegistryConfig) Evaluate() error {
	return errors.Join(
		z.CatErr(".addr", u.Addr.Evaluate()),
	)
}

func (u *RegistryConfig) Build(name string) (*remote.Registry, error) {
	reg, err := remote.NewRegistry(name)
	if err != nil {
		return nil, z.Err(err, "create repository client")
	}
	if u.Addr.Scheme() == "http" {
		reg.PlainHTTP = true
	}

	// TODO: from options
	reg.Client = &http.Client{
		Transport: otxhttp.NewTransport(http.DefaultTransport),
	}

	return reg, nil
}
