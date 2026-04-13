package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
)

// RSASigner signs data using RSA PKCS1v15 with SHA-256.
type RSASigner struct{}

func (s *RSASigner) Sign(data []byte, privateKeyDER []byte) ([]byte, error) {
	key, err := x509.ParsePKCS8PrivateKey(privateKeyDER)
	if err != nil {
		return nil, fmt.Errorf("parse RSA private key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected RSA private key, got %T", key)
	}

	hash := sha256.Sum256(data)
	return rsa.SignPKCS1v15(rand.Reader, rsaKey, crypto.SHA256, hash[:])
}

// GenerateRSAKeyPair generates a 2048-bit RSA key pair and returns DER-encoded bytes.
func GenerateRSAKeyPair() (privDER, pubDER []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate RSA key: %w", err)
	}

	privDER, err = x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal RSA private key: %w", err)
	}

	pubDER, err = x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal RSA public key: %w", err)
	}

	return privDER, pubDER, nil
}
