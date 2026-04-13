package persistence

import (
	"testing"
	"time"

	"github.com/alekstut/signing-service-challenge/domain"
)

func newTestDevice(id string) *domain.SignatureDevice {
	return &domain.SignatureDevice{
		ID:        id,
		Algorithm: domain.AlgorithmECC,
		Label:     "test",
		PrivateKey: []byte("priv"),
		PublicKey:  []byte("pub"),
		CreatedAt: time.Now(),
	}
}

func TestSave_FindByID_RoundTrip(t *testing.T) {
	repo := NewInMemoryRepository()
	device := newTestDevice("dev-1")

	if err := repo.Save(device); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	found, err := repo.FindByID("dev-1")
	if err != nil {
		t.Fatalf("FindByID() error: %v", err)
	}
	if found.ID != "dev-1" || found.Label != "test" {
		t.Fatalf("unexpected device: %+v", found)
	}
}

func TestSave_DuplicateID(t *testing.T) {
	repo := NewInMemoryRepository()
	device := newTestDevice("dev-1")

	if err := repo.Save(device); err != nil {
		t.Fatalf("first Save() error: %v", err)
	}
	if err := repo.Save(newTestDevice("dev-1")); err != domain.ErrDeviceAlreadyExists {
		t.Fatalf("expected ErrDeviceAlreadyExists, got %v", err)
	}
}

func TestSave_UpdateAfterSign(t *testing.T) {
	repo := NewInMemoryRepository()
	device := newTestDevice("dev-1")

	if err := repo.Save(device); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Simulate a signed device (counter > 0)
	device.SignatureCounter = 1
	device.LastSignature = "sig1"
	if err := repo.Save(device); err != nil {
		t.Fatalf("update Save() should succeed, got: %v", err)
	}

	found, _ := repo.FindByID("dev-1")
	if found.SignatureCounter != 1 {
		t.Fatalf("expected counter 1, got %d", found.SignatureCounter)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	_, err := repo.FindByID("nonexistent")
	if err != domain.ErrDeviceNotFound {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestFindAll_Empty(t *testing.T) {
	repo := NewInMemoryRepository()
	devices, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(devices))
	}
}

func TestFindAll_Multiple(t *testing.T) {
	repo := NewInMemoryRepository()
	for i := range 3 {
		repo.Save(newTestDevice(string(rune('a' + i))))
	}

	devices, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error: %v", err)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}
}

func TestSave_ReturnsCopy(t *testing.T) {
	repo := NewInMemoryRepository()
	device := newTestDevice("dev-1")
	repo.Save(device)

	// Mutate the returned entity
	found, _ := repo.FindByID("dev-1")
	found.Label = "mutated"
	found.PrivateKey[0] = 0xFF

	// Verify the stored entity is unchanged
	original, _ := repo.FindByID("dev-1")
	if original.Label != "test" {
		t.Fatal("stored device label was mutated")
	}
	if original.PrivateKey[0] == 0xFF {
		t.Fatal("stored device private key was mutated")
	}
}

func TestSave_StoresCopy(t *testing.T) {
	repo := NewInMemoryRepository()
	device := newTestDevice("dev-1")
	repo.Save(device)

	// Mutate the original after saving
	device.Label = "mutated"

	found, _ := repo.FindByID("dev-1")
	if found.Label != "test" {
		t.Fatal("store did not make a copy on Save")
	}
}
