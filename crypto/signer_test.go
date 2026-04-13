package crypto

import (
	"testing"

	"github.com/alekstut/signing-service-challenge/domain"
)

func TestNewSigner_RSA(t *testing.T) {
	s, err := NewSigner(domain.AlgorithmRSA)
	if err != nil {
		t.Fatalf("NewSigner(RSA) error: %v", err)
	}
	if _, ok := s.(*RSASigner); !ok {
		t.Fatalf("expected *RSASigner, got %T", s)
	}
}

func TestNewSigner_ECC(t *testing.T) {
	s, err := NewSigner(domain.AlgorithmECC)
	if err != nil {
		t.Fatalf("NewSigner(ECC) error: %v", err)
	}
	if _, ok := s.(*ECCSigner); !ok {
		t.Fatalf("expected *ECCSigner, got %T", s)
	}
}

func TestNewSigner_Unsupported(t *testing.T) {
	_, err := NewSigner("EdDSA")
	if err != domain.ErrUnsupportedAlgorithm {
		t.Fatalf("expected ErrUnsupportedAlgorithm, got %v", err)
	}
}

func TestGenerateKeyPair_RSA(t *testing.T) {
	priv, pub, err := GenerateKeyPair(domain.AlgorithmRSA)
	if err != nil {
		t.Fatalf("GenerateKeyPair(RSA) error: %v", err)
	}
	if len(priv) == 0 || len(pub) == 0 {
		t.Fatal("keys should not be empty")
	}
}

func TestGenerateKeyPair_ECC(t *testing.T) {
	priv, pub, err := GenerateKeyPair(domain.AlgorithmECC)
	if err != nil {
		t.Fatalf("GenerateKeyPair(ECC) error: %v", err)
	}
	if len(priv) == 0 || len(pub) == 0 {
		t.Fatal("keys should not be empty")
	}
}

func TestGenerateKeyPair_Unsupported(t *testing.T) {
	_, _, err := GenerateKeyPair("EdDSA")
	if err != domain.ErrUnsupportedAlgorithm {
		t.Fatalf("expected ErrUnsupportedAlgorithm, got %v", err)
	}
}
