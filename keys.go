package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

type ByteableKey interface {
	Bytes() []byte
}

func Base64Key(b ByteableKey) string {
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

var WireguardCurve = ecdh.X25519()

func NewWireguardPrivateKey() (*ecdh.PrivateKey, error) {
	randBytes := [32]byte{}
	rand.Read(randBytes[:])
	key, err := WireguardCurve.NewPrivateKey(randBytes[:])
	if err != nil {
		return nil, err
	}
	return key, nil
}

func WireguardPrivateKey(b64key string) (*ecdh.PrivateKey, error) {
	if b64key == "" {
		return nil, errors.New("cannot parse private key: no key specified")
	}
	keyBytes, err := base64.RawStdEncoding.DecodeString(b64key)
	if err != nil {
		return nil, err
	}
	key, err := WireguardCurve.NewPrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func WireguardPublicKey(b64key string) (*ecdh.PublicKey, error) {
	if b64key == "" {
		return nil, errors.New("cannot parse public key: no key specified")
	}
	keyBytes, err := base64.RawStdEncoding.DecodeString(b64key)
	if err != nil {
		return nil, err
	}
	key, err := WireguardCurve.NewPublicKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return key, nil

}
