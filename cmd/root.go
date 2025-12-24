package cmd

import (
	"github.com/lesomnus/xli"
)

func NewCmdRoot() *xli.Command {
	return &xli.Command{
		Name: "oras-get",

		Commands: xli.Commands{
			NewCmdServe(),
		},
	}
}
