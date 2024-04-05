package main

import (
	"errors"
	"fmt"
	"net"
	"path"
	"strings"

	_ "github.com/go-yaml/yaml"
)

const DEFAULT_SERVER_NAME = "wg-vlan"
const DEFAULT_LISTEN_PORT = 51820
const DEFAULT_NETWORK = "10.20.30.1/24"
const DEFAULT_KEEP_ALIVE = 25

type VLAN struct {
	PublicEndpoint string        `yaml:"public_endpoint"`
	KeepAlive      uint          `yaml:"keep_alive"`
	Server         VLANServer    `yaml:"server"`
	Clients        []*VLANClient `yaml:"clients"`
}

func (vlan VLAN) NextAddress() (*net.IP, error) {
	serverIP, vlanNetwork, err := net.ParseCIDR(vlan.Server.Network)
	if err != nil {
		return nil, err
	}

	takenIPs := []net.IP{serverIP}
	takenNets := []net.IPNet{}
	for _, client := range vlan.Clients {
		if strings.Contains(client.Network, "/") {
			clientIP, clientNet, err := net.ParseCIDR(client.Network)
			if err != nil {
				return nil, err
			}
			takenIPs = append(takenIPs, clientIP)
			takenNets = append(takenNets, *clientNet)
		} else {
			clientIP := net.ParseIP(client.Network)
			takenIPs = append(takenIPs, clientIP)
		}
	}

	return pickNextIP(*vlanNetwork, takenIPs, takenNets)
}

func (vlan VLAN) Validate() (vWarnings []string, vError error) {
	vErrors := []error{}
	if vlan.KeepAlive == 0 {
		vWarnings = append(vWarnings, "keep-alive is not set")
	}

	if vlan.PublicEndpoint == "" {
		vWarnings = append(vWarnings, "public endpoint not set")
	}

	srvWarnings, srvError := vlan.Server.Validate()
	if srvError != nil {
		vErrors = append(vErrors, fmt.Errorf("server: %w", srvError))
	}
	for _, warning := range srvWarnings {
		vWarnings = append(vWarnings, fmt.Sprintf("server: %s", warning))
	}

	uniqueClientNames := map[string]struct{}{}

	for idx, client := range vlan.Clients {
		clWarnings, clError := client.Validate()
		for _, warning := range clWarnings {
			vWarnings = append(vWarnings, fmt.Sprintf("client[%d]: %s", idx, warning))
		}
		if clError != nil {
			vErrors = append(vErrors, fmt.Errorf("client[%d]: %w", idx, clError))
		}
		if _, ok := uniqueClientNames[client.PeerName]; !ok && client.PeerName != "" {
			vErrors = append(vErrors, fmt.Errorf("client[%d]: non-unique client name", idx))
		}
	}

	if len(vErrors) > 0 {
		vError = fmt.Errorf("validation failed: %w", errors.Join(vErrors...))
	}
	return
}

func (vlan *VLAN) NewClient(name string, privateKeyBase64 string) (*VLANClient, error) {
	for _, client := range vlan.Clients {
		if client.PeerName == name {
			return nil, fmt.Errorf("name is already in use: %s", name)
		}
	}

	clientIP, err := vlan.NextAddress()
	if err != nil {
		return nil, err
	}

	if privateKeyBase64 == "" {
		privateKey, err := NewWireguardPrivateKey()
		if err != nil {
			return nil, err
		}
		privateKeyBase64 = KeyToBase64(privateKey)
	}

	client := &VLANClient{
		PeerName:   name,
		Network:    clientIP.String(),
		PrivateKey: privateKeyBase64,
	}

	if _, err := client.EnsurePublicKey(); err != nil {
		return nil, err
	}
	if _, err := client.EnsurePresharedKey(); err != nil {
		return nil, err
	}

	vlan.Clients = append(vlan.Clients, client)
	return client, nil
}

type VLANServer struct {
	InterfaceName  string            `yaml:"interface"`
	PeerName       string            `yaml:"peer_name"`
	ListenPort     uint              `yaml:"listen_port"`
	Network        string            `yaml:"network"`
	PrivateKey     string            `yaml:"private_key"`
	PublicKey      string            `yaml:"public_key"`
	ConfigINIPath  string            `yaml:"ini_path"`
	InterfaceExtra map[string]string `yaml:",inline"`
}

func (srv *VLANServer) EnsurePublicKey() (string, error) {
	if srv.PublicKey != "" {
		return srv.PublicKey, nil
	}
	key, err := WireguardPrivateKey(srv.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key '%s': %w", srv.PrivateKey, err)
	}
	srv.PublicKey = KeyToBase64(key.PublicKey())
	return srv.PublicKey, nil
}

func (srv *VLANServer) EnsurePath(yamlFile string) string {
	if srv.ConfigINIPath != "" {
		return srv.ConfigINIPath
	}
	srv.EnsureInterfaceName()
	srv.ConfigINIPath = path.Join(path.Dir(yamlFile), fmt.Sprintf("%s.conf", srv.InterfaceName))
	return srv.ConfigINIPath
}

func (srv *VLANServer) EnsureInterfaceName() string {
	if srv.InterfaceName == "" {
		srv.InterfaceName = DEFAULT_SERVER_NAME
	}
	return srv.InterfaceName
}

func (srv VLANServer) Validate() (vWarnings []string, vError error) {
	vErrors := []error{}
	if srv.PeerName == "" {
		vErrors = append(vErrors, errors.New("name not set"))
	}

	if srv.ListenPort == 0 {
		vErrors = append(vErrors, errors.New("listen port not set"))
	}

	if _, _, err := net.ParseCIDR(srv.Network); err != nil {
		vErrors = append(vErrors, fmt.Errorf("network invalid (%s): %w", srv.Network, err))
	}

	privateKey, pkeyErr := WireguardPrivateKey(srv.PrivateKey)
	if pkeyErr != nil {
		vErrors = append(vErrors, fmt.Errorf("private key invalid (%s): %w", srv.PrivateKey, pkeyErr))
	} else {
		expectPublicKey := KeyToBase64(privateKey.PublicKey())
		if srv.PublicKey != "" && srv.PublicKey != expectPublicKey {
			vErrors = append(vErrors, fmt.Errorf("public key mismatch: got '%s', expected '%s'", srv.PublicKey, expectPublicKey))
		}
	}
	if srv.ConfigINIPath == "" {
		vWarnings = append(vWarnings, "config INI path unset")
	}

	if len(vErrors) > 0 {
		vError = fmt.Errorf("validation failed: %w", errors.Join(vErrors...))
	}
	return
}

type VLANClient struct {
	PeerName       string            `yaml:"peer_name"`
	Network        string            `yaml:"network"`
	PrivateKey     string            `yaml:"private_key"`
	PublicKey      string            `yaml:"public_key"`
	PresharedKey   string            `yaml:"preshared_key"`
	ConfigINIPath  string            `yaml:"ini_path"`
	InterfaceExtra map[string]string `yaml:",inline"`
}

func (cl *VLANClient) EnsurePublicKey() (string, error) {
	if cl.PublicKey != "" {
		return cl.PublicKey, nil
	}
	key, err := WireguardPrivateKey(cl.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("invalid private key '%s': %w", cl.PrivateKey, err)
	}
	cl.PublicKey = KeyToBase64(key.PublicKey())
	return cl.PublicKey, nil
}

func (cl *VLANClient) EnsurePresharedKey() (string, error) {
	if cl.PresharedKey != "" {
		return cl.PresharedKey, nil
	}
	key, err := NewWireguardPrivateKey()
	if err != nil {
		return "", fmt.Errorf("failed generating preshared key: %w", err)
	}
	cl.PresharedKey = KeyToBase64(key)
	return cl.PresharedKey, nil

}

func (cl *VLANClient) EnsurePath(yamlFile string) string {
	if cl.ConfigINIPath != "" {
		return cl.ConfigINIPath
	}
	cl.ConfigINIPath = path.Join(path.Dir(yamlFile), "wg-clients", fmt.Sprintf("%s.conf", cl.PeerName))
	return cl.ConfigINIPath
}

func (cl VLANClient) Validate() (vWarnings []string, vError error) {
	vErrors := []error{}
	if cl.PeerName == "" {
		vErrors = append(vErrors, fmt.Errorf("client name unset"))
	}
	if _, _, err := net.ParseCIDR(cl.Network); err != nil {
		vErrors = append(vErrors, fmt.Errorf("client network invalid (%s): %w", cl.Network, err))
	}

	expectPublicKey := ""
	if cl.PrivateKey == "" {
		vWarnings = append(vWarnings, "client private key unset; will not be able to generate client config")
		if cl.PublicKey == "" {
			vErrors = append(vErrors, errors.New("client keys both unset"))
		}
	} else if privateKey, err := WireguardPrivateKey(cl.PrivateKey); err != nil {
		vErrors = append(vErrors, fmt.Errorf("client private key invalid: %s", cl.PrivateKey))
	} else {
		expectPublicKey = KeyToBase64(privateKey.PublicKey())
	}

	if cl.PublicKey != "" && cl.PublicKey != expectPublicKey {
		vErrors = append(vErrors, fmt.Errorf("client public key mismatch: got '%s', expected '%s'", cl.PublicKey, expectPublicKey))
	}

	if cl.ConfigINIPath == "" {
		vWarnings = append(vWarnings, "config INI path unset")
	}

	if len(vErrors) > 0 {
		vError = fmt.Errorf("validation failed: %w", errors.Join(vErrors...))
	}

	return
}

func DefaultVLAN(yamlPath string) (*VLAN, error) {
	privateKey, err := NewWireguardPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed generating new private key: %w", err)
	}

	vlan := &VLAN{
		PublicEndpoint: "",
		KeepAlive:      DEFAULT_KEEP_ALIVE,
		Server: VLANServer{
			PeerName:   DEFAULT_SERVER_NAME,
			ListenPort: DEFAULT_LISTEN_PORT,
			Network:    DEFAULT_NETWORK,
			PrivateKey: KeyToBase64(privateKey),
			PublicKey:  KeyToBase64(privateKey.PublicKey()),
		},
	}

	vlan.Server.EnsureInterfaceName()
	vlan.Server.EnsurePath(yamlPath)
	if _, err := vlan.Server.EnsurePublicKey(); err != nil {
		return nil, fmt.Errorf("failed generating public key: %w", err)
	}

	return vlan, nil
}
