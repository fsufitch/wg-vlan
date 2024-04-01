package main

import (
	"crypto/ecdh"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

type GenerateCommand struct {
	fName       *cli.StringFlag
	fAddress    *cli.StringFlag
	fPort       *cli.UintFlag
	fPrivateKey *cli.StringFlag
}

func mustHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	return hostname
}

func NewGenerateCommand() GenerateCommand {
	return GenerateCommand{
		fName: &cli.StringFlag{
			Name:    "name",
			Aliases: []string{"n"},
			Usage:   "name to use for the current host (for comment/organization purposes)",
			Value:   mustHostname(),
		},
		fAddress: &cli.StringFlag{
			Name:    "address",
			Aliases: []string{"ip", "a", "i"},
			Usage:   "network address of this peer; include a netmask to define the network size",
			Value:   "10.22.6.1/24",
		},
		fPort: &cli.UintFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "port to listen on",
			Value:   51820,
		},
		fPrivateKey: &cli.StringFlag{
			Name:        "key",
			Aliases:     []string{"k", "private-key"},
			Usage:       "128 bits as base64, to be used as a private key",
			DefaultText: "generate a new one",
			Value:       "",
		},
	}
}

func (c *GenerateCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "generate",
		Aliases:     []string{"gen"},
		Description: "generate a new empty Wireguard peer configuration to STDOUT",
		Args:        true,
		ArgsUsage:   "FILE",
		Action:      c.Action,
		Flags:       []cli.Flag{c.fAddress, c.fPort, c.fPrivateKey, c.fName},
	}

}

func (c GenerateCommand) Action(ctx *cli.Context) error {
	if ctx.Args().Len() > 0 {
		return cli.Exit(fmt.Errorf("extraneous arguments received: %v", ctx.Args()), 1)
	}

	iniFile, _ := NewWireguardInterfaceIni(nil)
	iniFile.Interface().SetName(c.fName.Value)

	addr, err := parseSingleIPNet(c.fAddress.Value)
	if err != nil {
		return cli.Exit(fmt.Sprintf("failed to parse address '%s': %v", c.fAddress.Value, err), 1)
	}
	iniFile.Interface().SetAddress(*addr)

	iniFile.Interface().SetListenPort(c.fPort.Value)

	var pk *ecdh.PrivateKey
	if c.fPrivateKey.Value == "" {
		pk, err = NewWireguardPrivateKey()
	} else {
		pk, err = WireguardPrivateKey(c.fPrivateKey.Value)
	}

	if err != nil {
		return cli.Exit(fmt.Sprintf("invalid private key '%s': %v", c.fAddress.Value, err), 1)
	}
	iniFile.Interface().SetPrivateKey(pk)
	iniFile.Prune()
	iniFile.WriteTo(os.Stdout)
	fmt.Fprintln(os.Stdout)
	return nil
}
