package api

import (
	"errors"

	"github.com/google/uuid"
)

// CreateDeviceRequest represents the request body for creating a signature device.
type CreateDeviceRequest struct {
	ID        string `json:"id"`
	Algorithm string `json:"algorithm"`
	Label     string `json:"label,omitempty"`
}

func (r *CreateDeviceRequest) Validate() error {
	if _, err := uuid.Parse(r.ID); err != nil {
		return errors.New("id must be a valid UUID")
	}
	if r.Algorithm == "" {
		return errors.New("algorithm is required")
	}
	return nil
}

// SignTransactionRequest represents the request body for signing transaction data.
type SignTransactionRequest struct {
	Data string `json:"data"`
}

func (r *SignTransactionRequest) Validate() error {
	if r.Data == "" {
		return errors.New("data is required")
	}
	return nil
}
