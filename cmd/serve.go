package cmd

import (
	"context"
	"net/http"

	"github.com/lesomnus/oras-get/og"
	"github.com/lesomnus/xli"
)

func NewCmdServe() *xli.Command {
	return &xli.Command{
		Name: "serve",

		Handler: xli.OnRun(func(ctx context.Context, cmd *xli.Command, next xli.Next) error {
			server := og.NewServer("registry:5000")

			s := http.Server{
				Addr:    "localhost:5001",
				Handler: server,
			}
			if err := s.ListenAndServe(); err != nil {
				return err
			}
			return next(ctx)
		}),
	}
}
