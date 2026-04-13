package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"testing"
)

func TestECC_GenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("GenerateECCKeyPair() error: %v", err)
	}
	if len(priv) == 0 {
		t.Fatal("private key is empty")
	}
	if len(pub) == 0 {
		t.Fatal("public key is empty")
	}
}

func TestECC_SignAndVerify(t *testing.T) {
	priv, pub, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("GenerateECCKeyPair() error: %v", err)
	}

	signer := &ECCSigner{}
	data := []byte("test transaction data")

	sig, err := signer.Sign(data, priv)
	if err != nil {
		t.Fatalf("Sign() error: %v", err)
	}
	if len(sig) == 0 {
		t.Fatal("signature is empty")
	}

	// Verify with public key
	pubKey, err := x509.ParsePKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("ParsePKIXPublicKey() error: %v", err)
	}
	ecPub, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatalf("expected *ecdsa.PublicKey, got %T", pubKey)
	}

	hash := sha256.Sum256(data)
	if !ecdsa.VerifyASN1(ecPub, hash[:], sig) {
		t.Fatal("signature verification failed")
	}
}

func TestECC_SignNonDeterministic(t *testing.T) {
	priv, _, err := GenerateECCKeyPair()
	if err != nil {
		t.Fatalf("GenerateECCKeyPair() error: %v", err)
	}

	signer := &ECCSigner{}
	data := []byte("same data")

	sig1, _ := signer.Sign(data, priv)
	sig2, _ := signer.Sign(data, priv)

	// ECDSA is non-deterministic — same input produces different signatures
	if string(sig1) == string(sig2) {
		t.Fatal("ECDSA signatures should be non-deterministic")
	}
}

func TestECC_SignInvalidKey(t *testing.T) {
	signer := &ECCSigner{}
	_, err := signer.Sign([]byte("data"), []byte("not a key"))
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
