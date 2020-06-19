/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package csp

import (
	"crypto"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cetcxinlian/cryptogm/x509"
	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/bccsp/factory"
	"github.com/hyperledger/fabric/bccsp/signer"
	"github.com/pkg/errors"
)

// LoadPrivateKey loads a private key from file in keystorePath
func LoadPrivateKey(keystorePath string) (bccsp.Key, crypto.Signer, error) {
	var err error
	var priv bccsp.Key
	var s crypto.Signer

	opts := &factory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &factory.SwOpts{
			HashFamily: "SHA2",
			SecLevel:   256,

			FileKeystore: &factory.FileKeystoreOpts{
				KeyStorePath: keystorePath,
			},
		},
	}

	csp, err := factory.GetBCCSPFromOpts(opts)
	if err != nil {
		return nil, nil, err
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, "_sk") {
			rawKey, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			block, _ := pem.Decode(rawKey)
			if block == nil {
				return errors.Errorf("%s: wrong PEM encoding", path)
			}
			priv, err = csp.KeyImport(block.Bytes, &bccsp.ECDSAPrivateKeyImportOpts{Temporary: true})
			if err != nil {
				return err
			}

			s, err = signer.New(csp, priv)
			if err != nil {
				return err
			}

			return nil
		}
		return nil
	}

	err = filepath.Walk(keystorePath, walkFunc)
	if err != nil {
		return nil, nil, err
	}

	return priv, s, err
}

// GeneratePrivateKey creates a private key and stores it in keystorePath
func GeneratePrivateKey(keystorePath string, isGm bool) (bccsp.Key,
	crypto.Signer, error) {

	var err error
	var priv bccsp.Key
	var s crypto.Signer

	opts := &factory.FactoryOpts{
		ProviderName: "SW",
		SwOpts: &factory.SwOpts{
			HashFamily: "SHA2",
			SecLevel:   256,

			FileKeystore: &factory.FileKeystoreOpts{
				KeyStorePath: keystorePath,
			},
		},
	}
	csp, err := factory.GetBCCSPFromOpts(opts)
	if err == nil {
		// generate a key
		if isGm {
			priv, err = csp.KeyGen(&bccsp.SM2KeyGenOpts{Temporary: false})
			if err == nil {
				// create a crypto.Signer
				s, err = signer.New(csp, priv)
			}
		} else {
			priv, err = csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{Temporary: false})
			if err == nil {
				// create a crypto.Signer
				s, err = signer.New(csp, priv)
			}
		}
	}
	return priv, s, err
}

func GetPublicKey(priv bccsp.Key) (interface{}, error) {
	// get the public key
	pubKey, err := priv.PublicKey()
	if err != nil {
		return nil, err
	}
	// marshal to bytes
	pubKeyBytes, err := pubKey.Bytes()
	if err != nil {
		return nil, err
	}
	// unmarshal using pkix
	return x509.ParsePKIXPublicKey(pubKeyBytes)
}
