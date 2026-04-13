package api

import (
	"encoding/base64"
	"time"

	"github.com/alekstut/signing-service-challenge/domain"
)

// DeviceResponse is the API representation of a signature device.
// Private keys are deliberately excluded.
type DeviceResponse struct {
	ID               string `json:"id"`
	Algorithm        string `json:"algorithm"`
	Label            string `json:"label,omitempty"`
	SignatureCounter int    `json:"signature_counter"`
	PublicKey        string `json:"public_key"`
	CreatedAt        string `json:"created_at"`
}

// NewDeviceResponse maps a domain entity to an API response.
func NewDeviceResponse(d *domain.SignatureDevice) DeviceResponse {
	return DeviceResponse{
		ID:               d.ID,
		Algorithm:        string(d.Algorithm),
		Label:            d.Label,
		SignatureCounter: d.SignatureCounter,
		PublicKey:        base64.StdEncoding.EncodeToString(d.PublicKey),
		CreatedAt:        d.CreatedAt.Format(time.RFC3339),
	}
}

// SignatureResponse is returned after signing transaction data.
type SignatureResponse struct {
	Signature  string `json:"signature"`
	SignedData string `json:"signed_data"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error string `json:"error"`
}
