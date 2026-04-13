package crypto

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"testing"
)

func TestRSA_GenerateKeyPair(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair() error: %v", err)
	}
	if len(priv) == 0 {
		t.Fatal("private key is empty")
	}
	if len(pub) == 0 {
		t.Fatal("public key is empty")
	}
}

func TestRSA_SignAndVerify(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair() error: %v", err)
	}

	signer := &RSASigner{}
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
	rsaPub, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		t.Fatalf("expected *rsa.PublicKey, got %T", pubKey)
	}

	hash := sha256.Sum256(data)
	if err := rsa.VerifyPKCS1v15(rsaPub, crypto.SHA256, hash[:], sig); err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

func TestRSA_SignDeterministic(t *testing.T) {
	priv, _, err := GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair() error: %v", err)
	}

	signer := &RSASigner{}
	data := []byte("same data")

	sig1, _ := signer.Sign(data, priv)
	sig2, _ := signer.Sign(data, priv)

	// RSA PKCS1v15 is deterministic — same input produces same signature
	if string(sig1) != string(sig2) {
		t.Fatal("RSA PKCS1v15 signatures should be deterministic")
	}
}

func TestRSA_SignInvalidKey(t *testing.T) {
	signer := &RSASigner{}
	_, err := signer.Sign([]byte("data"), []byte("not a key"))
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
