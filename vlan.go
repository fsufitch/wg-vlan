package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/go-yaml/yaml"
	"gopkg.in/ini.v1"
)

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
	serverIP, vlanNetwork, err := parseCIDR(vlan.Server.Network)
	if err != nil {
		return nil, err
	}

	takenIPs := []net.IP{serverIP}
	takenNets := []net.IPNet{}
	for _, client := range vlan.Clients {
		clientIP, clientNet, err := parseCIDR(client.Network)
		if err != nil {
			return nil, err
		}
		takenIPs = append(takenIPs, clientIP)
		takenNets = append(takenNets, *clientNet)
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
		if _, ok := uniqueClientNames[client.PeerName]; ok && client.PeerName != "" {
			vErrors = append(vErrors, fmt.Errorf("client[%d]: non-unique client name", idx))
		}
	}

	if len(vErrors) > 0 {
		vError = fmt.Errorf("validation failed: %w", errors.Join(vErrors...))
	}
	return
}

func (vlan *VLAN) NewClient(name string, privateKeyBase64 string) (*VLANClient, error) {
	if name == "" {
		return nil, errors.New("client may not have an empty name")
	}

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

func (vlan *VLAN) NewClientPublic(name string, publicKeyBase64 string) (*VLANClient, error) {
	if name == "" {
		return nil, errors.New("client may not have an empty name")
	}

	for _, client := range vlan.Clients {
		if client.PeerName == name {
			return nil, fmt.Errorf("name is already in use: %s", name)
		}
	}

	clientIP, err := vlan.NextAddress()
	if err != nil {
		return nil, err
	}

	client := &VLANClient{
		PeerName:  name,
		Network:   clientIP.String(),
		PublicKey: publicKeyBase64,
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
	PeerName       string            `yaml:"peer_name"`
	ListenPort     uint              `yaml:"listen_port"`
	Network        string            `yaml:"network"`
	PrivateKey     string            `yaml:"private_key"`
	PublicKey      string            `yaml:"public_key"`
	InterfaceExtra map[string]string `yaml:"extra"`
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
	InterfaceExtra map[string]string `yaml:"extra"`
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

func (cl VLANClient) CIDR() (net.IP, *net.IPNet, error) {
	if strings.Contains(cl.Network, "/") {
		// When it's a proper CIDR, this is easy
		return net.ParseCIDR(cl.Network)
	}

	// When no CIDR, add /32; IPv6 has unknown behavior
	return net.ParseCIDR(cl.Network + "/32")
}

func (cl VLANClient) Validate() (vWarnings []string, vError error) {
	vErrors := []error{}
	if cl.PeerName == "" {
		vErrors = append(vErrors, fmt.Errorf("client name unset"))
	}

	if _, _, err := cl.CIDR(); err != nil {
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

	if cl.PresharedKey == "" {
		vWarnings = append(vWarnings, "client preshared key unset; this is unsafe")
	}

	if len(vErrors) > 0 {
		vError = fmt.Errorf("validation failed: %w", errors.Join(vErrors...))
	}

	return
}

func (vlan VLAN) WriteTo(path string) error {
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(fp)
	if err := enc.Encode(vlan); err != nil {
		return err
	}

	if err := enc.Close(); err != nil {
		return err
	}
	if err := fp.Close(); err != nil {
		return err
	}
	return nil
}

func (vlan VLAN) ServerIni() (*ini.File, error) {
	iniFile := ini.Empty(ini.LoadOptions{AllowNonUniqueSections: true})

	iniFile.Section("Interface").Comment = fmt.Sprintf("# VLAN Server: %s", vlan.Server.PeerName)

	iniFile.Section("Interface").Key("Address").SetValue(vlan.Server.Network)
	iniFile.Section("Interface").Key("ListenPort").SetValue(fmt.Sprintf("%d", vlan.Server.ListenPort))
	iniFile.Section("Interface").Key("PrivateKey").SetValue(vlan.Server.PrivateKey)

	for _, client := range vlan.Clients {
		sec, _ := iniFile.NewSection("Peer")
		sec.Comment = fmt.Sprintf("# VLAN Client: %s", client.PeerName)

		clientIP, err := ensureIPWithCIDR(client.Network)
		if err != nil {
			return nil, fmt.Errorf("peer failed '%s': %w", client.PeerName, err)
		}
		sec.Key("AllowedIPs").SetValue(clientIP)

		publicKey, err := client.EnsurePublicKey()
		if err != nil {
			return nil, fmt.Errorf("peer failed '%s': %w", client.PeerName, err)
		}
		sec.Key("PublicKey").SetValue(publicKey)

		if client.PresharedKey != "" {
			sec.Key("PresharedKey").SetValue(client.PresharedKey)
		}

		if vlan.KeepAlive != 0 {
			sec.Key("PersistentKeepalive").SetValue(fmt.Sprintf("%d", vlan.KeepAlive))
		}
	}

	return iniFile, nil
}

func (vlan VLAN) ClientIni(clientName string) (*ini.File, error) {
	var client *VLANClient
	for _, cl := range vlan.Clients {
		if cl.PeerName == clientName {
			client = cl
			break
		}
	}
	if client == nil {
		return nil, fmt.Errorf("no such client: %s", clientName)
	}
	if client.PrivateKey == "" {
		return nil, fmt.Errorf("client has no private key defined: %s", clientName)
	}

	iniFile := ini.Empty(ini.LoadOptions{AllowNonUniqueSections: true})
	iniFile.Section("Interface").Comment = fmt.Sprintf("# VLAN Client: %s", client.PeerName)
	clientIP, err := ensureIPWithCIDR(client.Network)
	if err != nil {
		return nil, fmt.Errorf("client '%s' had invalid network '%s': %w", clientName, client.Network, err)
	}
	iniFile.Section("Interface").Key("Address").SetValue(clientIP)
	iniFile.Section("Interface").Key("PrivateKey").SetValue(client.PrivateKey)

	serverSection, _ := iniFile.NewSection("Peer")
	serverSection.Comment = fmt.Sprintf("# VLAN Server: %s", vlan.Server.PeerName)

	if vlan.PublicEndpoint == "" {
		return nil, errors.New("vlan has no configured public endpoint")
	}
	serverSection.Key("Endpoint").SetValue(vlan.PublicEndpoint)

	serverIP, err := ensureIPWithCIDR(vlan.Server.Network)
	if err != nil {
		return nil, fmt.Errorf("server had invalid network '%s': %w", vlan.Server.Network, err)
	}
	serverSection.Key("AllowedIPs").SetValue(serverIP)

	serverPublicKey, err := vlan.Server.EnsurePublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get server public key: %w", err)
	}
	serverSection.Key("PublicKey").SetValue(serverPublicKey)

	if client.PresharedKey != "" {
		serverSection.Key("PresharedKey").SetValue(client.PresharedKey)
	}

	if vlan.KeepAlive != 0 {
		serverSection.Key("PersistentKeepalive").SetValue(fmt.Sprintf("%d", vlan.KeepAlive))
	}

	return iniFile, nil
}

func VLANFromFile(path string, warningLogger *log.Logger) (*VLAN, error) {
	fp, err := os.Open(path)
	defer func() {
		if err := fp.Close(); err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("failed to open config file (%s): %w", path, err)
	}

	vlan := &VLAN{}
	if err := yaml.NewDecoder(fp).Decode(vlan); err != nil {
		return nil, fmt.Errorf("failed to decode config file (%s): %w", path, err)
	}

	vWarnings, vError := vlan.Validate()
	for _, w := range vWarnings {
		if warningLogger != nil {
			warningLogger.Printf("config warning: %s", w)
		}
	}
	if vError != nil {
		return nil, vError
	}

	return vlan, nil
}

func parseCIDR(address string) (net.IP, *net.IPNet, error) {
	if strings.Contains(address, "/") {
		// When it's a proper CIDR, this is easy
		return net.ParseCIDR(address)
	}

	// When no CIDR, add /32; IPv6 has unknown behavior
	return net.ParseCIDR(address + "/32")
}

func ensureIPWithCIDR(address string) (string, error) {
	ip, ipNet, err := parseCIDR(address)
	if err != nil {
		return "", err
	}
	ipNet.IP = ip
	return ipNet.String(), nil
}
