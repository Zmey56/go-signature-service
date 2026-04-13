package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
)

// ECCSigner signs data using ECDSA with SHA-256.
type ECCSigner struct{}

func (s *ECCSigner) Sign(data []byte, privateKeyDER []byte) ([]byte, error) {
	key, err := x509.ParsePKCS8PrivateKey(privateKeyDER)
	if err != nil {
		return nil, fmt.Errorf("parse ECC private key: %w", err)
	}
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected ECDSA private key, got %T", key)
	}

	hash := sha256.Sum256(data)
	return ecdsa.SignASN1(rand.Reader, ecKey, hash[:])
}

// GenerateECCKeyPair generates a P-256 ECDSA key pair and returns DER-encoded bytes.
func GenerateECCKeyPair() (privDER, pubDER []byte, err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ECC key: %w", err)
	}

	privDER, err = x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ECC private key: %w", err)
	}

	pubDER, err = x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ECC public key: %w", err)
	}

	return privDER, pubDER, nil
}
