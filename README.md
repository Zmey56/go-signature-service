# Signature Service

A Go REST API service for managing signature devices and signing transaction data with cryptographic chain integrity.

## Quick Start

```bash
make run          # build and start server on :8080
make test         # run all tests
make test-race    # run all tests with race detector
make lint         # run go vet
```

## API

| Method | Endpoint                    | Description               |
| ------ | --------------------------- | ------------------------- |
| `POST` | `/api/v1/devices`           | Create a signature device |
| `GET`  | `/api/v1/devices`           | List all devices          |
| `GET`  | `/api/v1/devices/{id}`      | Get a device by ID        |
| `POST` | `/api/v1/devices/{id}/sign` | Sign transaction data     |
| `GET`  | `/health`                   | Health check              |

### Create Device

```bash
curl -X POST localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{"id":"550e8400-e29b-41d4-a716-446655440000","algorithm":"ECC","label":"POS Terminal 1"}'
```

Supported algorithms: `ECC` (ECDSA P-256) and `RSA` (RSA-2048 PKCS1v15).

### Sign Transaction

```bash
curl -X POST localhost:8080/api/v1/devices/550e8400-e29b-41d4-a716-446655440000/sign \
  -H "Content-Type: application/json" \
  -d '{"data":"payment:100EUR"}'
```

Response:

```json
{
    "signature": "<base64_encoded_signature>",
    "signed_data": "<counter>_<data>_<last_signature_base64>"
}
```

## Architecture

```
cmd/server/       Entry point, dependency wiring, graceful shutdown
domain/           Core types, interfaces, business logic (zero external deps)
crypto/           RSA and ECDSA signer implementations (Strategy pattern)
service/          Orchestration layer with per-device mutex concurrency control
persistence/      In-memory repository with deep-copy isolation
api/              HTTP handlers, DTOs, middleware, routing (stdlib only)
```

### Key Design Decisions

**Per-device mutex in the service layer.** The `signature_counter` must be strictly monotonically increasing without gaps. A global lock would serialize all signing across unrelated devices. A repository-level lock would create a TOCTOU gap between read and update. The per-device mutex in the service layer locks the entire read-sign-increment-save cycle for one device while allowing other devices to operate concurrently.

**Deep copies in the repository.** The `InMemoryDeviceRepository` stores and returns clones of `SignatureDevice` structs. This prevents callers from accidentally mutating shared state through pointer aliasing — the same guarantee a SQL database provides naturally. This makes the in-memory implementation a true drop-in for a future database backend.

**Stdlib-only routing.** Go 1.22+ `http.ServeMux` supports method-based pattern matching (`"POST /api/v1/devices/{id}/sign"`), eliminating the need for external routers. The only external dependency is `github.com/google/uuid`.

**`SecuredData()` on the domain entity.** The signed data format (`<counter>_<data>_<last_signature>`) is domain logic that depends only on the device's own state, so it lives as a method on `SignatureDevice`.

**No private keys in API responses.** The `DeviceResponse` DTO deliberately excludes `private_key`. Only the `public_key` is exposed via the API.

### Extensibility

To add a new signing algorithm (e.g., Ed25519):

1. Implement the `domain.Signer` interface in `crypto/`
2. Add a `GenerateXxxKeyPair()` function
3. Register in `crypto/signer.go` (signers map + GenerateKeyPair switch)
4. Add the algorithm constant to `domain/device.go`

No changes to the service layer, API handlers, or persistence layer.

## Testing

The test suite includes **40+ tests** across 4 packages:

- **Crypto tests**: Generate key pair, sign data, verify signature with public key (roundtrip proof). Tests RSA determinism and ECDSA non-determinism.
- **Persistence tests**: CRUD operations + copy isolation (mutate returned entity, verify stored entity unchanged).
- **Service tests**: Chain integrity (3 sequential signatures verify chaining), base case (first signature uses `base64(deviceID)`), counter monotonicity, **100-goroutine concurrent signing** on a single device.
- **HTTP integration tests**: Full request/response cycle via `httptest`. All status codes (201, 200, 400, 404, 409). End-to-end 5-signature chain test via HTTP.

All tests pass with `go test -race ./...`.

## AI Tools Disclosure

This project was developed with assistance from **Claude Code** (Anthropic's CLI tool, Claude Opus 4.6 model) and **Genini**. They were used for:

- Architecture design and implementation planning
- Test case design and implementation
- help with README documentation
