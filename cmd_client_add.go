package main

import (
	"github.com/urfave/cli/v2"
)

type ClientAddCommand struct {
	fConfigFile string
	fClientName string
	fPublicKey  string
}

func (c *ClientAddCommand) Command() *cli.Command {
	return &cli.Command{
		Name:    "client-add",
		Aliases: []string{"add"},
		Args:    false,
		Action:  c.Action,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:        "vlan-config",
				Aliases:     []string{"config", "c"},
				Usage:       "YAML config file to write to",
				Required:    true,
				Destination: &c.fConfigFile,
			},
			&cli.StringFlag{
				Name:        "client-name",
				Aliases:     []string{"name", "n"},
				Usage:       "name of client to add",
				Required:    true,
				Destination: &c.fClientName,
			},
			&cli.StringFlag{
				Name:        "public-key",
				Aliases:     []string{"pub"},
				Usage:       "public key of the client",
				DefaultText: "generate a new private/public pair",
				Destination: &c.fPublicKey,
			},
		},
	}
}

func (c *ClientAddCommand) Action(ctx *cli.Context) error {
	cLog := getLogger(ctx)

	vlan, err := VLANFromFile(c.fConfigFile)
	if err != nil {
		cLog.Fatalf("error: %s", err.Error())
	}
	vWarnings, vError := vlan.Validate()
	for _, w := range vWarnings {
		cLog.Printf("config warning: %s", w)
	}
	if vError != nil {
		cLog.Fatalf("config error: %s", vError.Error())
	}

	var newClient *VLANClient
	if c.fPublicKey == "" {
		newClient, err = vlan.NewClient(c.fClientName, "")
	} else {
		newClient, err = vlan.NewClientPublic(c.fClientName, c.fPublicKey)
	}
	if err != nil {
		cLog.Fatalf("failed to create client: %s", err.Error())
	}

	newClient.EnsurePath(c.fConfigFile)

	cLog.Printf("successfully created client: %s - %s", newClient.PeerName, newClient.Network)

	if err := vlan.WriteTo(c.fConfigFile); err != nil {
		cLog.Fatalf("error: failed to write config file: %s", err.Error())
	}

	cLog.Printf("wrote configuration to: %s", c.fConfigFile)

	return nil
}
