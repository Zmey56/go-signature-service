package domain

import (
	"encoding/base64"
	"fmt"
	"time"
)

// Algorithm represents a supported signing algorithm.
type Algorithm string

const (
	AlgorithmRSA Algorithm = "RSA"
	AlgorithmECC Algorithm = "ECC"
)

// ParseAlgorithm validates and returns an Algorithm or an error.
func ParseAlgorithm(s string) (Algorithm, error) {
	switch Algorithm(s) {
	case AlgorithmRSA:
		return AlgorithmRSA, nil
	case AlgorithmECC:
		return AlgorithmECC, nil
	default:
		return "", ErrUnsupportedAlgorithm
	}
}

// SignatureDevice is the core domain entity representing a device that can sign data.
type SignatureDevice struct {
	ID               string
	Algorithm        Algorithm
	Label            string
	SignatureCounter int
	LastSignature    string // base64-encoded; empty when no signature has been created yet
	PrivateKey       []byte // PKCS8 DER-encoded
	PublicKey        []byte // PKIX DER-encoded
	CreatedAt        time.Time
}

// SecuredData builds the string to be signed per the specification.
// Format: "<counter>_<data_to_be_signed>_<last_signature_base64>"
// Base case (counter == 0): uses base64(device.ID) instead of last_signature.
func (d *SignatureDevice) SecuredData(dataToBeSigned string) string {
	lastSig := d.LastSignature
	if d.SignatureCounter == 0 {
		lastSig = base64.StdEncoding.EncodeToString([]byte(d.ID))
	}
	return fmt.Sprintf("%d_%s_%s", d.SignatureCounter, dataToBeSigned, lastSig)
}

// Clone returns a deep copy of the device.
func (d *SignatureDevice) Clone() *SignatureDevice {
	clone := *d
	clone.PrivateKey = make([]byte, len(d.PrivateKey))
	copy(clone.PrivateKey, d.PrivateKey)
	clone.PublicKey = make([]byte, len(d.PublicKey))
	copy(clone.PublicKey, d.PublicKey)
	return &clone
}
