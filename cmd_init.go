package main

import (
	"errors"
	"os"

	"github.com/urfave/cli/v2"
)

type InitializeCommand struct {
	fConfigFile string
	fEndpoint   string
	fNetwork    string
	fPeerName   string
	fListenPort uint
	fPrivateKey string
	fClients    cli.StringSlice
}

func (c *InitializeCommand) Command() *cli.Command {
	return &cli.Command{
		Name:        "init",
		Description: "initialize a new Wireguard VLAN server configuration to the specified file",
		Args:        false,
		Action:      c.Action,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:        "vlan-config",
				Aliases:     []string{"f"},
				Usage:       "YAML config file to write to",
				Required:    true,
				Destination: &c.fConfigFile,
			},
			&cli.StringFlag{
				Name:        "name",
				Aliases:     []string{"n"},
				Usage:       "peer name of the server",
				Value:       "wg-vlan",
				Destination: &c.fPeerName,
			},
			&cli.StringFlag{
				Name:        "endpoint",
				Aliases:     []string{"e"},
				Usage:       "public endpoint for clients to connect to",
				Destination: &c.fEndpoint,
			},
			&cli.StringFlag{
				Name:        "network",
				Aliases:     []string{"net"},
				Usage:       "CIDR address/mask of the VLAN subnet",
				Value:       DEFAULT_NETWORK,
				Destination: &c.fNetwork,
			},
			&cli.UintFlag{
				Name:        "port",
				Aliases:     []string{"p"},
				Usage:       "port to listen on",
				Value:       DEFAULT_LISTEN_PORT,
				Destination: &c.fListenPort,
			},
			&cli.StringFlag{
				Name:        "private-key",
				Aliases:     []string{"k"},
				Usage:       "private key to use",
				DefaultText: "generate a new one",
				Destination: &c.fPrivateKey,
			},
			&cli.StringSliceFlag{
				Name:        "client",
				Usage:       "auto-generate a client with this name",
				Destination: &c.fClients,
			},
		},
	}
}

func (c *InitializeCommand) Action(ctx *cli.Context) error {
	cLog := getLogger(ctx)

	if c.fConfigFile == "" {
		cLog.Fatalf("error: YAML path required")
	}

	if c.fPrivateKey == "" {
		cLog.Printf("generating private key")
		pk, err := NewWireguardPrivateKey()
		if err != nil {
			cLog.Fatalf("failed generating a private key: %v", err)
		}
		cLog.Printf("generated new private key; public=%s", KeyToBase64(pk.PublicKey()))
		c.fPrivateKey = KeyToBase64(pk)
	}

	vlan := VLAN{
		PublicEndpoint: c.fEndpoint,
		KeepAlive:      DEFAULT_KEEP_ALIVE,
		Server: VLANServer{
			PeerName:   c.fPeerName,
			ListenPort: c.fListenPort,
			Network:    c.fNetwork,
			PrivateKey: c.fPrivateKey,
		},
	}

	if _, err := vlan.Server.EnsurePublicKey(); err != nil {
		cLog.Fatalf("error: %v", err)
	}

	for _, clientName := range c.fClients.Value() {
		_, err := vlan.NewClient(clientName, "")
		if err != nil {
			cLog.Fatalf("error: %v", err)
		}
	}

	if _, err := os.Stat(c.fConfigFile); !errors.Is(err, os.ErrNotExist) {
		cLog.Fatalf("error: config already exists: %s", c.fConfigFile)
	}

	if err := vlan.WriteTo(c.fConfigFile); err != nil {
		cLog.Fatalf("error: failed to write config file: %s", err.Error())
	}

	cLog.Printf("wrote configuration to: %s", c.fConfigFile)

	return nil
}
