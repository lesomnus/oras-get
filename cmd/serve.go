package cmd

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/lesomnus/oras-get/config"
	"github.com/lesomnus/otx/log"
	"github.com/lesomnus/xli"
	"github.com/lesomnus/z"
)

func NewCmdServe() *xli.Command {
	return &xli.Command{
		Name: "serve",

		Handler: xli.OnRun(func(ctx context.Context, cmd *xli.Command, next xli.Next) error {
			l := log.From(ctx)
			c := config.From(ctx)

			r, err := c.NewRouter()
			if err != nil {
				return z.Err(err, "build router")
			}

			s := http.Server{
				Addr:    c.Server.Addr.HostPort(),
				Handler: r,
			}

			l.Info("listen", slog.String("addr", s.Addr))
			lis, err := c.Server.Addr.Listen()
			if err != nil {
				return z.Err(err, "listen %s", s.Addr)
			}

			l.Info("serve")
			if err := s.Serve(lis); err != nil {
				return err
			}
			return next(ctx)
		}),
	}
}
