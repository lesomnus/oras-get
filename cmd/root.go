package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/lesomnus/oras-get/config"
	"github.com/lesomnus/otx"
	"github.com/lesomnus/otx/log"
	"github.com/lesomnus/xli"
	"github.com/lesomnus/xli/flg"
	"github.com/lesomnus/xli/frm"
	"github.com/lesomnus/z"
)

func NewCmdRoot() *xli.Command {
	return &xli.Command{
		Name: "oras-get",

		Flags: flg.Flags{
			&flg.String{Name: "conf"},
		},

		Commands: xli.Commands{
			NewCmdVersion(),
			NewCmdServe(),
		},

		Handler: xli.Chain(
			xli.RequireSubcommand(),
			xli.OnRunPass(func(ctx context.Context, cmd *xli.Command, next xli.Next) error {
				if f := frm.From(ctx); frm.HasSeq(f.Next(), "version") {
					return next(ctx)
				}

				conf_path := "oras-get.yaml"
				flg.VisitP(cmd, "conf", &conf_path)

				use_default_conf := false
				c, err := config.FromPath(conf_path)
				if err != nil {
					if !errors.Is(err, os.ErrNotExist) || cmd.Flags.Get("conf").Count() > 0 {
						return fmt.Errorf("read config at %q: %w", conf_path, err)
					}

					c = &config.Config{}
					use_default_conf = true
					conf_path = "!default"
				}
				if err := c.Evaluate(); err != nil {
					return fmt.Errorf("evaluate the config at %q: %w", conf_path, err)
				}

				otx_, err := c.Otel.Build(ctx)
				if err != nil {
					return fmt.Errorf("build OpenTelemetry context from the config %q: %w", conf_path, err)
				}

				defer otx_.Shutdown(ctx)
				if err := otx_.Start(ctx); err != nil {
					return z.Err(err, "start OpenTelemetry")
				}

				ctx = config.Into(ctx, c)
				ctx = otx.Into(ctx, otx_)

				l := log.From(ctx)
				if use_default_conf {
					l.Info("use default config")
				} else {
					l.Info("config is read", slog.String("path", conf_path))
				}
				return next(ctx)
			}),
		),
	}
}
