package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alekstut/signing-service-challenge/domain"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS signature_devices (
    id                TEXT PRIMARY KEY,
    algorithm         TEXT NOT NULL,
    label             TEXT NOT NULL DEFAULT '',
    signature_counter INTEGER NOT NULL DEFAULT 0,
    last_signature    TEXT NOT NULL DEFAULT '',
    private_key       BYTEA NOT NULL,
    public_key        BYTEA NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

// PostgresDeviceRepository stores signature devices in PostgreSQL.
type PostgresDeviceRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed device repository.
// It creates the required table if it doesn't exist.
func NewPostgresRepository(ctx context.Context, connStr string) (*PostgresDeviceRepository, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if _, err := pool.Exec(ctx, createTableSQL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}

	return &PostgresDeviceRepository{pool: pool}, nil
}

// Close closes the connection pool.
func (r *PostgresDeviceRepository) Close() {
	r.pool.Close()
}

func (r *PostgresDeviceRepository) Save(device *domain.SignatureDevice) error {
	ctx := context.Background()

	if device.SignatureCounter == 0 {
		// New device — INSERT. Unique constraint violation means duplicate.
		_, err := r.pool.Exec(ctx, `
			INSERT INTO signature_devices (id, algorithm, label, signature_counter, last_signature, private_key, public_key, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
			device.ID, string(device.Algorithm), device.Label,
			device.SignatureCounter, device.LastSignature,
			device.PrivateKey, device.PublicKey, device.CreatedAt,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return domain.ErrDeviceAlreadyExists
			}
			return fmt.Errorf("insert device: %w", err)
		}
		return nil
	}

	// Existing device — UPDATE counter and last_signature after signing.
	_, err := r.pool.Exec(ctx, `
		UPDATE signature_devices
		SET signature_counter = $1, last_signature = $2
		WHERE id = $3
	`, device.SignatureCounter, device.LastSignature, device.ID)
	if err != nil {
		return fmt.Errorf("update device: %w", err)
	}
	return nil
}

func (r *PostgresDeviceRepository) FindByID(id string) (*domain.SignatureDevice, error) {
	ctx := context.Background()

	var d domain.SignatureDevice
	var alg string
	err := r.pool.QueryRow(ctx, `
		SELECT id, algorithm, label, signature_counter, last_signature, private_key, public_key, created_at
		FROM signature_devices WHERE id = $1
	`, id).Scan(
		&d.ID, &alg, &d.Label, &d.SignatureCounter,
		&d.LastSignature, &d.PrivateKey, &d.PublicKey, &d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrDeviceNotFound
		}
		return nil, fmt.Errorf("find device: %w", err)
	}

	d.Algorithm = domain.Algorithm(alg)
	return &d, nil
}

func (r *PostgresDeviceRepository) FindAll() ([]*domain.SignatureDevice, error) {
	ctx := context.Background()

	rows, err := r.pool.Query(ctx, `
		SELECT id, algorithm, label, signature_counter, last_signature, private_key, public_key, created_at
		FROM signature_devices ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []*domain.SignatureDevice
	for rows.Next() {
		var d domain.SignatureDevice
		var alg string
		if err := rows.Scan(
			&d.ID, &alg, &d.Label, &d.SignatureCounter,
			&d.LastSignature, &d.PrivateKey, &d.PublicKey, &d.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		d.Algorithm = domain.Algorithm(alg)
		devices = append(devices, &d)
	}

	if devices == nil {
		devices = []*domain.SignatureDevice{}
	}
	return devices, rows.Err()
}
