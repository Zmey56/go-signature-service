package domain

import "errors"

var (
	ErrDeviceNotFound       = errors.New("signature device not found")
	ErrDeviceAlreadyExists  = errors.New("signature device already exists")
	ErrUnsupportedAlgorithm = errors.New("unsupported signing algorithm")
)
