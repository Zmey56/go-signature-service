package persistence

import (
	"sync"

	"github.com/alekstut/signing-service-challenge/domain"
)

// InMemoryDeviceRepository stores signature devices in memory.
// All returned entities are deep copies to prevent callers from mutating shared state.
type InMemoryDeviceRepository struct {
	mu      sync.RWMutex
	devices map[string]*domain.SignatureDevice
}

// NewInMemoryRepository creates a new in-memory device repository.
func NewInMemoryRepository() *InMemoryDeviceRepository {
	return &InMemoryDeviceRepository{
		devices: make(map[string]*domain.SignatureDevice),
	}
}

func (r *InMemoryDeviceRepository) Save(device *domain.SignatureDevice) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.devices[device.ID]
	if exists && existing.SignatureCounter == 0 && device.SignatureCounter == 0 {
		return domain.ErrDeviceAlreadyExists
	}

	r.devices[device.ID] = device.Clone()
	return nil
}

func (r *InMemoryDeviceRepository) FindByID(id string) (*domain.SignatureDevice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	device, ok := r.devices[id]
	if !ok {
		return nil, domain.ErrDeviceNotFound
	}
	return device.Clone(), nil
}

func (r *InMemoryDeviceRepository) FindAll() ([]*domain.SignatureDevice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*domain.SignatureDevice, 0, len(r.devices))
	for _, d := range r.devices {
		result = append(result, d.Clone())
	}
	return result, nil
}
