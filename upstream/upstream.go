package upstream

import (
	"errors"
	"net/http"

	"github.com/lesomnus/oras-get/addr"
	"github.com/lesomnus/z"
	"oras.land/oras-go/v2/registry/remote"
)

type Upstream struct {
	Addr addr.Http
	// TLS Verify?
	// Prefix?
	// Auth?
	// Forwarded Headers?
}

func (u *Upstream) Evaluate() error {
	return errors.Join(
		z.CatErr(".addr", u.Addr.Evaluate()),
	)
}

func (u *Upstream) Build(name string) (*remote.Repository, error) {
	repo, err := remote.NewRepository(name)
	if err != nil {
		return nil, z.Err(err, "create repository client")
	}
	if u.Addr.Scheme() == "http" {
		repo.PlainHTTP = true
	}

	// TODO: from options
	repo.Client = http.DefaultClient

	return repo, nil
}
