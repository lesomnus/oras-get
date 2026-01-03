package config

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lesomnus/oras-get/addr"
	"github.com/lesomnus/oras-get/og/upstream"
	"github.com/lesomnus/otx/otxhttp"
	"github.com/lesomnus/z"
	"oras.land/oras-go/v2/registry/remote"
)

type RegistryConfig struct {
	Addr     addr.Http
	Redirect bool
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

func (u *RegistryConfig) Build() (upstream.Upstream, error) {
	domain := u.Addr.HostPort()
	reg, err := remote.NewRegistry(domain)
	if err != nil {
		return upstream.Upstream{}, z.Err(err, "create repository client")
	}

	scheme := u.Addr.Scheme()
	switch scheme {
	case "":
		scheme = "https"
	case "http":
		reg.PlainHTTP = true
	case "https":
		reg.PlainHTTP = false
	case "unix":
		if u.Redirect {
			return upstream.Upstream{}, errors.New("redirect not supported for unix socket")
		}
	default:
		return upstream.Upstream{}, fmt.Errorf("unsupported scheme: %q", scheme)
	}

	// TODO: from options
	reg.Client = &http.Client{
		Transport: otxhttp.NewTransport(http.DefaultTransport),
	}

	return upstream.Upstream{
		Scheme:   scheme,
		Domain:   domain,
		Registry: reg,
		Redirect: u.Redirect,
	}, nil
}
