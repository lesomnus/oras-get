package upstream

import (
	"context"

	"github.com/lesomnus/oras-get/refs"
	"oras.land/oras-go/v2/registry"
)

type Upstream struct {
	Scheme   string
	Domain   string
	Registry registry.Registry
	Redirect bool
}

func (u Upstream) Repository(ctx context.Context, ref refs.Ref) (Repository, error) {
	repo, err := u.Registry.Repository(ctx, ref.Repo())
	if err != nil {
		return Repository{}, err
	}

	return Repository{
		Upstream:   u,
		Reference:  refs.WithDomain(ref, u.Domain),
		Repository: repo,
	}, nil
}
