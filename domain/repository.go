package domain

// DeviceRepository defines storage operations for signature devices.
type DeviceRepository interface {
	Save(device *SignatureDevice) error
	FindByID(id string) (*SignatureDevice, error)
	FindAll() ([]*SignatureDevice, error)
}
