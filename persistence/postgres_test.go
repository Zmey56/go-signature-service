package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/alekstut/signing-service-challenge/domain"
)

func setupPostgresRepo(t *testing.T) *PostgresDeviceRepository {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()

	ctr, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { testcontainers.TerminateContainer(ctr) })

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	repo, err := NewPostgresRepository(ctx, connStr)
	if err != nil {
		t.Fatalf("create postgres repo: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	return repo
}

func newPgTestDevice(id string) *domain.SignatureDevice {
	return &domain.SignatureDevice{
		ID:         id,
		Algorithm:  domain.AlgorithmECC,
		Label:      "test",
		PrivateKey: []byte("priv-key-data"),
		PublicKey:  []byte("pub-key-data"),
		CreatedAt:  time.Now().Truncate(time.Microsecond), // PostgreSQL precision
	}
}

func TestPostgres_Save_FindByID_RoundTrip(t *testing.T) {
	repo := setupPostgresRepo(t)
	device := newPgTestDevice("pg-dev-1")

	if err := repo.Save(device); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	found, err := repo.FindByID("pg-dev-1")
	if err != nil {
		t.Fatalf("FindByID() error: %v", err)
	}
	if found.ID != "pg-dev-1" {
		t.Fatalf("expected ID pg-dev-1, got %s", found.ID)
	}
	if found.Label != "test" {
		t.Fatalf("expected label 'test', got %s", found.Label)
	}
	if found.Algorithm != domain.AlgorithmECC {
		t.Fatalf("expected ECC, got %s", found.Algorithm)
	}
	if string(found.PrivateKey) != "priv-key-data" {
		t.Fatalf("private key mismatch")
	}
	if string(found.PublicKey) != "pub-key-data" {
		t.Fatalf("public key mismatch")
	}
}

func TestPostgres_Save_DuplicateID(t *testing.T) {
	repo := setupPostgresRepo(t)

	if err := repo.Save(newPgTestDevice("pg-dev-1")); err != nil {
		t.Fatalf("first Save() error: %v", err)
	}
	if err := repo.Save(newPgTestDevice("pg-dev-1")); err != domain.ErrDeviceAlreadyExists {
		t.Fatalf("expected ErrDeviceAlreadyExists, got %v", err)
	}
}

func TestPostgres_Save_UpdateAfterSign(t *testing.T) {
	repo := setupPostgresRepo(t)
	device := newPgTestDevice("pg-dev-1")

	if err := repo.Save(device); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	device.SignatureCounter = 1
	device.LastSignature = "sig1-base64"
	if err := repo.Save(device); err != nil {
		t.Fatalf("update Save() should succeed, got: %v", err)
	}

	found, _ := repo.FindByID("pg-dev-1")
	if found.SignatureCounter != 1 {
		t.Fatalf("expected counter 1, got %d", found.SignatureCounter)
	}
	if found.LastSignature != "sig1-base64" {
		t.Fatalf("expected last_signature 'sig1-base64', got %s", found.LastSignature)
	}
}

func TestPostgres_FindByID_NotFound(t *testing.T) {
	repo := setupPostgresRepo(t)
	_, err := repo.FindByID("nonexistent")
	if err != domain.ErrDeviceNotFound {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestPostgres_FindAll_Empty(t *testing.T) {
	repo := setupPostgresRepo(t)
	devices, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(devices))
	}
}

func TestPostgres_FindAll_Multiple(t *testing.T) {
	repo := setupPostgresRepo(t)
	for _, id := range []string{"a", "b", "c"} {
		if err := repo.Save(newPgTestDevice(id)); err != nil {
			t.Fatalf("Save(%s) error: %v", id, err)
		}
	}

	devices, err := repo.FindAll()
	if err != nil {
		t.Fatalf("FindAll() error: %v", err)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}
}

func TestPostgres_FindAll_OrderedByCreatedAt(t *testing.T) {
	repo := setupPostgresRepo(t)

	// Create devices with different timestamps
	for i, id := range []string{"first", "second", "third"} {
		d := newPgTestDevice(id)
		d.CreatedAt = time.Now().Add(time.Duration(i) * time.Second).Truncate(time.Microsecond)
		repo.Save(d)
	}

	devices, _ := repo.FindAll()
	if devices[0].ID != "first" || devices[2].ID != "third" {
		t.Fatalf("expected ordered by created_at, got: %s, %s, %s",
			devices[0].ID, devices[1].ID, devices[2].ID)
	}
}

func TestPostgres_MultipleSignatures(t *testing.T) {
	repo := setupPostgresRepo(t)
	device := newPgTestDevice("pg-dev-sign")
	repo.Save(device)

	for i := range 10 {
		device.SignatureCounter = i + 1
		device.LastSignature = "sig-" + string(rune('a'+i))
		if err := repo.Save(device); err != nil {
			t.Fatalf("Save() at counter %d: %v", i+1, err)
		}
	}

	found, _ := repo.FindByID("pg-dev-sign")
	if found.SignatureCounter != 10 {
		t.Fatalf("expected counter 10, got %d", found.SignatureCounter)
	}
}

// TestPostgres_ImplementsInterface verifies at compile time that
// PostgresDeviceRepository satisfies domain.DeviceRepository.
var _ domain.DeviceRepository = (*PostgresDeviceRepository)(nil)
