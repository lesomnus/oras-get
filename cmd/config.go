package cmd

import (
	"context"

	"github.com/lesomnus/oras-get/cmd/config"
	"github.com/lesomnus/xli"
	"gopkg.in/yaml.v3"
)

func NewCmdConfig() *xli.Command {
	return &xli.Command{
		Name:  "config",
		Brief: "manage configuration",

		Commands: xli.Commands{
			NewCmdConfigDump(),
		},
	}
}

func NewCmdConfigDump() *xli.Command {
	return &xli.Command{
		Name:  "dump",
		Brief: "dump current configuration",

		Handler: xli.Chain(
			xli.OnRun(func(ctx context.Context, cmd *xli.Command, next xli.Next) error {
				c := config.From(ctx)

				if err := yaml.NewEncoder(cmd).Encode(c); err != nil {
					return err
				}
				return next(ctx)
			}),
		),
	}
}
