package main

import (
	"os"

	"github.com/go-yaml/yaml"
	"github.com/urfave/cli/v2"
)

type InitializeCommand struct {
	fConfigFile string
	fEndpoint   string
	fNetwork    string
	fInterface  string
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
				Name:        "f",
				Aliases:     []string{"config", "config-file"},
				Usage:       "YAML config file to write to",
				Required:    true,
				Destination: &c.fConfigFile,
			},
			&cli.StringFlag{
				Name:        "e",
				Aliases:     []string{"endpoint"},
				Usage:       "public endpoint for clients to connect to",
				Destination: &c.fEndpoint,
			},
			&cli.StringFlag{
				Name:        "net",
				Aliases:     []string{"network"},
				Usage:       "CIDR address/mask of the VLAN subnet",
				Value:       DEFAULT_NETWORK,
				Destination: &c.fNetwork,
			},
			&cli.StringFlag{
				Name:        "i",
				Aliases:     []string{"interface"},
				Usage:       "name of the interface to use",
				Value:       DEFAULT_SERVER_NAME,
				Destination: &c.fInterface,
			},
			&cli.UintFlag{
				Name:        "p",
				Aliases:     []string{"port"},
				Usage:       "port to listen on",
				Value:       DEFAULT_LISTEN_PORT,
				Destination: &c.fListenPort,
			},
			&cli.StringFlag{
				Name:        "k",
				Aliases:     []string{"private-key"},
				Usage:       "private key to use",
				DefaultText: "generate a new one",
				Destination: &c.fPrivateKey,
			},
			&cli.StringSliceFlag{
				Name:        "c",
				Aliases:     []string{"client"},
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
		cLog.Printf("private=%s public=%s", KeyToBase64(pk), KeyToBase64(pk.PublicKey()))
		c.fPrivateKey = KeyToBase64(pk)
	}

	vlan := VLAN{
		PublicEndpoint: c.fEndpoint,
		KeepAlive:      DEFAULT_KEEP_ALIVE,
		Server: VLANServer{
			InterfaceName: c.fInterface,
			PeerName:      c.fInterface,
			ListenPort:    c.fListenPort,
			Network:       c.fNetwork,
			PrivateKey:    c.fPrivateKey,
		},
	}

	vlan.Server.EnsurePath(c.fConfigFile)
	if _, err := vlan.Server.EnsurePublicKey(); err != nil {
		cLog.Fatalf("error: %v", err)
	}

	for _, clientName := range c.fClients.Value() {
		client, err := vlan.NewClient(clientName, "")
		if err != nil {
			cLog.Fatalf("error: %v", err)
		}
		client.EnsurePath(c.fConfigFile)
	}

	fp, err := os.Create(c.fConfigFile)
	if err != nil {
		cLog.Fatalf("error: %v", err)
	}
	enc := yaml.NewEncoder(fp)
	if err := enc.Encode(vlan); err != nil {
		cLog.Fatalf("error: %v", err)
	}

	enc.Close()
	fp.Close()

	cLog.Printf("wrote configuration to: %s", c.fConfigFile)

	return nil
}
