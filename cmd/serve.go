package cmd

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/lesomnus/oras-get/cmd/config"
	"github.com/lesomnus/otx"
	"github.com/lesomnus/otx/log"
	"github.com/lesomnus/otx/otxhttp"
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

			h := http.Handler(r)
			h = otxhttp.BoundaryLogger()(h)
			h = otxhttp.NewMiddleware(otx.From(ctx), "serve")(h)
			s := http.Server{Handler: h}

			addr := c.Server.Addr.HostPort()

			l.Info("listen", slog.String("addr", addr))
			lis, err := c.Server.Addr.Listen()
			if err != nil {
				return z.Err(err, "listen %s", addr)
			}

			l.Info("serve")
			if err := s.Serve(lis); err != nil {
				return err
			}
			return next(ctx)
		}),
	}
}
