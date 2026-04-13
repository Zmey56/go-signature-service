package domain

// Signer signs data using a specific algorithm and private key.
type Signer interface {
	Sign(data []byte, privateKeyDER []byte) (signature []byte, err error)
}
