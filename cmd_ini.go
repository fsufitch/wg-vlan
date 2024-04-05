package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

type PrintIniCommand struct {
	fConfigFile   string
	fServerOutput bool
	fClientOutput string
	fFormat       string
}

func (c *PrintIniCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "ini",
		Description: "print Wireguard INI files",
		Args:        false,
		Action:      c.Action,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:        "vlan-config",
				Aliases:     []string{"config", "c"},
				Usage:       "YAML config file to write to",
				Required:    true,
				Destination: &c.fConfigFile,
			},
			&cli.BoolFlag{
				Name:        "server",
				Aliases:     []string{"s"},
				Usage:       "print the server INI; mutually exclusive with --client",
				Destination: &c.fServerOutput,
			},
			&cli.StringFlag{
				Name:        "client",
				Usage:       "print a client's INI; mutually exclusive with --server",
				Destination: &c.fClientOutput,
			},
			&ChoicesFlag{
				StringFlag: cli.StringFlag{
					Name:        "format",
					Aliases:     []string{"f"},
					Usage:       "output format to use",
					Destination: &c.fFormat,
					Value:       "text",
				},
				Choices: []string{"text", "qr"},
			},
		},
	}
}

func (c *PrintIniCommand) Action(ctx *cli.Context) error {
	fmt.Printf("format '%+v'\n", c.fFormat)

	fmt.Printf("asdf '%+v'\n", ctx.String("format"))

	return nil
}
