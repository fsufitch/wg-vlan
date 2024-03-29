package main

import (
	"crypto/ecdh"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type WireguardInterfaceIni struct {
	// Functionality based on https://github.com/pirate/wireguard-docs/blob/master/README.md#Config-Reference
	*ini.File
}

func NewWireguardInterfaceIni(reader io.ReadCloser) (*WireguardInterfaceIni, error) {
	loadOptions := ini.LoadOptions{AllowNonUniqueSections: true}
	if reader == nil {
		ifIni := &WireguardInterfaceIni{
			ini.Empty(loadOptions),
		}
		ifIni.NewSection("Interface")
		return ifIni, nil
	}
	iniFile, err := ini.LoadSources(loadOptions, reader)
	if err != nil {
		return nil, err
	}
	return &WireguardInterfaceIni{iniFile}, nil
}

func (iniFile *WireguardInterfaceIni) Prune() {
	pruneEmptyKeys(iniFile.Interface().Section)
	for idx, peer := range iniFile.Peers() {
		pruneEmptyKeys(peer.Section)
		if len(peer.Keys()) == 0 {
			iniFile.DeleteSectionWithIndex("Peer", idx)
		}
	}
}

func (iniFile *WireguardInterfaceIni) Interface() *WireguardInterfaceIni_Interface {
	return &WireguardInterfaceIni_Interface{iniFile.Section("Interface")}
}

func (iniFile *WireguardInterfaceIni) Peers() (peers []*WireguardInterfaceIni_Peer) {
	if !iniFile.HasSection("Peer") {
		return peers
	}
	sections, _ := iniFile.SectionsByName("Peer")
	for _, sec := range sections {
		peers = append(peers, &WireguardInterfaceIni_Peer{sec})
	}
	return peers
}

func (iniFile *WireguardInterfaceIni) Peer(name string) *WireguardInterfaceIni_Peer {
	for _, peer := range iniFile.Peers() {
		if peer.Name() == name {
			return peer
		}
	}
	newSec, _ := iniFile.NewSection("Peer")
	return &WireguardInterfaceIni_Peer{newSec}
}

type WireguardInterfaceIni_Interface struct {
	*ini.Section
}

func (sec *WireguardInterfaceIni_Interface) Name() string {
	return getSectionName(sec.Section)
}

func (sec *WireguardInterfaceIni_Interface) SetName(name string) {
	setSectionName(sec.Section, name)
}

func (sec *WireguardInterfaceIni_Interface) Address() (*net.IPNet, error) {
	return parseSingleIPNet(sec.Key("Address").MustString(""))
}

func (sec *WireguardInterfaceIni_Interface) SetAddress(ipWithCIDR net.IPNet) {
	sec.Key("Address").SetValue(ipWithCIDR.String())
}

func (sec *WireguardInterfaceIni_Interface) ListenPort() uint {
	return sec.Key("ListenPort").MustUint()
}

func (sec *WireguardInterfaceIni_Interface) SetListenPort(p uint) {
	sec.Key("ListenPort").SetValue(fmt.Sprintf("%d", p))
}

func (sec *WireguardInterfaceIni_Interface) PrivateKey() (*ecdh.PrivateKey, error) {
	return WireguardPrivateKey(sec.Key("PrivateKey").MustString(""))
}

func (sec *WireguardInterfaceIni_Interface) SetPrivateKey(privateKey *ecdh.PrivateKey) {
	sec.Key("PrivateKey").SetValue(Base64Key(privateKey))
}

type WireguardInterfaceIni_Peer struct {
	*ini.Section
}

func (peer *WireguardInterfaceIni_Peer) Name() string {
	return getSectionName(peer.Section)
}

func (peer *WireguardInterfaceIni_Peer) SetName(name string) {
	setSectionName(peer.Section, name)
}

func (peer *WireguardInterfaceIni_Peer) AllowedIPs() (*net.IPNet, error) {
	return parseSingleIPNet(peer.Key("AllowedIPs").Value())
}

func (peer *WireguardInterfaceIni_Peer) SetAllowedIPs(ipWithCIDR net.IPNet) {
	peer.Key("AllowedIPs").SetValue(ipWithCIDR.String())
}

func (peer *WireguardInterfaceIni_Peer) Endpoint() (string, int) {
	host, portStr, _ := strings.Cut(peer.Key("Endpoint").MustString(""), ":")
	port := 0
	if portStr != "" {
		port, _ = strconv.Atoi(portStr)
	}
	if port == 0 {
		return "", 0
	}
	return host, port
}

func (peer *WireguardInterfaceIni_Peer) SetEndpoint(host string, port int) {
	if host == "" || port == 0 {
		peer.DeleteKey("Endpoint")
	}
	peer.Key("Endpoint").SetValue(fmt.Sprintf("%s:%d", host, port))
}

func (peer *WireguardInterfaceIni_Peer) PublicKey() (*ecdh.PublicKey, error) {
	return WireguardPublicKey(peer.Key("PublicKey").Value())
}

func (peer *WireguardInterfaceIni_Peer) SetPublicKey(publicKey *ecdh.PublicKey) {
	peer.Key("PublicKey").SetValue(Base64Key(publicKey))
}

func (peer *WireguardInterfaceIni_Peer) PresharedKey() (*ecdh.PublicKey, error) {
	return WireguardPublicKey(peer.Key("PresharedKey").Value())
}

func (peer *WireguardInterfaceIni_Peer) SetPresharedKey(presharedKey *ecdh.PublicKey) {
	peer.Key("PresharedKey").SetValue(Base64Key(presharedKey))
}
