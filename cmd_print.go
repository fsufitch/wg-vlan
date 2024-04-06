package main

import (
	"fmt"
	"strings"

	"github.com/skip2/go-qrcode"
	"github.com/urfave/cli/v2"
	"gopkg.in/ini.v1"
)

type PrintIniCommand struct {
	fConfigFile   string
	fServerOutput bool
	fClientOutput string
	fFormat       string
}

func (c *PrintIniCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "print",
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
	switch c.fFormat {
	case "text":
		return c.printText(ctx)
	case "qr":
		return c.printQR(ctx)
	}
	return fmt.Errorf("unknown format: '%s'", c.fFormat)
}

func (c *PrintIniCommand) printText(ctx *cli.Context) error {
	cLog := getLogger(ctx)

	if c.fServerOutput && c.fClientOutput != "" {
		cLog.Fatalf("cannot output both server and client configs at once")
	}
	if !c.fServerOutput && c.fClientOutput == "" {
		cLog.Fatalf("must specify either --server or --client")
	}

	vlan, err := VLANFromFile(c.fConfigFile, cLog)
	if err != nil {
		cLog.Fatalf("error reading config: %s", err.Error())
	}

	var iniFile *ini.File

	if c.fServerOutput {
		iniFile, err = vlan.ServerIni()
	} else {
		iniFile, err = vlan.ClientIni(c.fClientOutput)
	}
	if err != nil {
		cLog.Fatalf("error building ini: %s", err.Error())
	}

	if _, err := iniFile.WriteTo(ctx.App.Writer); err != nil {
		cLog.Fatalf("error writing ini: %s", err.Error())
	}

	return nil
}

func (c *PrintIniCommand) printQR(ctx *cli.Context) error {
	cLog := getLogger(ctx)

	if c.fServerOutput && c.fClientOutput != "" {
		cLog.Fatalf("cannot output both server and client configs at once")
	}
	if !c.fServerOutput && c.fClientOutput == "" {
		cLog.Fatalf("must specify either --server or --client")
	}

	vlan, err := VLANFromFile(c.fConfigFile, cLog)
	if err != nil {
		cLog.Fatalf("error reading config: %s", err.Error())
	}

	var iniFile *ini.File

	if c.fServerOutput {
		iniFile, err = vlan.ServerIni()
	} else {
		iniFile, err = vlan.ClientIni(c.fClientOutput)
	}
	if err != nil {
		cLog.Fatalf("error building ini: %s", err.Error())
	}

	buf := strings.Builder{}

	if _, err := iniFile.WriteTo(&buf); err != nil {
		cLog.Fatalf("error writing ini: %s", err.Error())
	}

	qr, err := qrcode.New(buf.String(), qrcode.Low)
	if err != nil {
		cLog.Fatalf("error constructing QR: %s", err.Error())
	}
	fmt.Fprintln(ctx.App.Writer, qr.ToSmallString(true))
	return nil
}
