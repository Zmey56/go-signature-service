package crypto

import (
	"github.com/alekstut/signing-service-challenge/domain"
)

var signers = map[domain.Algorithm]domain.Signer{
	domain.AlgorithmRSA: &RSASigner{},
	domain.AlgorithmECC: &ECCSigner{},
}

// NewSigner returns the Signer for the given algorithm.
func NewSigner(alg domain.Algorithm) (domain.Signer, error) {
	s, ok := signers[alg]
	if !ok {
		return nil, domain.ErrUnsupportedAlgorithm
	}
	return s, nil
}

// GenerateKeyPair generates a new key pair for the given algorithm.
// Returns DER-encoded private key (PKCS8) and public key (PKIX).
func GenerateKeyPair(alg domain.Algorithm) (privDER, pubDER []byte, err error) {
	switch alg {
	case domain.AlgorithmRSA:
		return GenerateRSAKeyPair()
	case domain.AlgorithmECC:
		return GenerateECCKeyPair()
	default:
		return nil, nil, domain.ErrUnsupportedAlgorithm
	}
}
