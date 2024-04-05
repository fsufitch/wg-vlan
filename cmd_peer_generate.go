package main

import (
	"os"

	"github.com/urfave/cli/v2"
)

type PeerGenerateCommand struct {
	fServerIni  *cli.PathFlag
	fAddress    *cli.StringFlag
	fPort       *cli.UintFlag
	fPrivateKey *cli.StringFlag
}

func NewPeerGenerateCommand() PeerGenerateCommand {
	return PeerGenerateCommand{
		fServerIni: &cli.PathFlag{
			Name:    "server-ini-file",
			Aliases: []string{"f", "config"},
			Usage:   "path to the server INI configuration to use",
		},
		fAddress: &cli.StringFlag{
			Name:        "address",
			Aliases:     []string{"ip", "i", "allowed-ips", "a"},
			Usage:       "address or CIDR block this peer is allowed to claim and receive traffic from",
			DefaultText: "pick a single IP from the server's IP block",
		},
		fPort: &cli.UintFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Usage:   "port for the peer to listen on",
			Value:   51820,
		},
		fPrivateKey: &cli.StringFlag{
			Name:        "key",
			Aliases:     []string{"k", "private-key"},
			Usage:       "128 bits as base64, to be used as a private key for the peer",
			DefaultText: "generate",
			Value:       "",
		},
	}
}

func (c *PeerGenerateCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "peer-generate",
		Aliases:     []string{"peer-gen"},
		Description: "create and add a new peer to the network",
		Args:        true,
		ArgsUsage:   "NAME",
		Flags:       []cli.Flag{c.fServerIni, c.fAddress, c.fPort, c.fPrivateKey},
		Action:      c.Action,
	}
}

func (c PeerGenerateCommand) Action(ctx *cli.Context) error {
	cLog := getLogger(ctx)
	peerName := ctx.Args().First()
	if peerName == "" {
		cLog.Fatalf("error: peer name required")
	}
	if ctx.Args().Len() > 1 {
		cLog.Fatalf("extraneous arguments received: %+v", ctx.Args().Slice()[1:])
	}

	serverIniPath := c.fServerIni.FilePath
	if _, err := os.Stat(serverIniPath); err != nil {
		cLog.Fatalf("error finding server ini (%s): %+v %+v", serverIniPath, err, c.fServerIni.HasBeenSet)
	}

	return nil
}
