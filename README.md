# Signature Service

A Go REST API service for managing signature devices and signing transaction data with cryptographic chain integrity.

## Quick Start

```bash
# Run locally
make run                # build and start server on :8080

# Run in Docker
make docker-run         # build image (~25 MB) and run on :8080

# Testing
make test               # unit tests — fast, no Docker required
make test-race          # unit tests with race detector
make test-integration   # PostgreSQL integration tests via testcontainers-go (requires Docker)

# Other
make docker-build       # build Docker image only
make lint               # run go vet
make clean              # remove build artifacts
```

### Prerequisites

- Go 1.22+ (for stdlib routing with method patterns)
- Docker (for `make test-integration` and `make docker-run`)

## API

| Method | Endpoint | Description | Success |
|--------|----------|-------------|---------|
| `POST` | `/api/v1/devices` | Create a signature device | `201` |
| `GET` | `/api/v1/devices` | List all devices | `200` |
| `GET` | `/api/v1/devices/{id}` | Get a device by ID | `200` |
| `POST` | `/api/v1/devices/{id}/sign` | Sign transaction data | `200` |
| `GET` | `/health` | Health check | `200` |

### Error Codes

| Code | When |
|------|------|
| `400` | Invalid JSON, missing fields, unsupported algorithm |
| `404` | Device not found |
| `409` | Device with this ID already exists |
| `500` | Internal server error |

### Create Device

```bash
curl -X POST localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{"id":"550e8400-e29b-41d4-a716-446655440000","algorithm":"ECC","label":"POS Terminal 1"}'
```

Supported algorithms: `ECC` (ECDSA P-256) and `RSA` (RSA-2048 PKCS1v15). The `label` field is optional.

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
  "signed_data": "0_payment:100EUR_NTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAw"
}
```

The `signed_data` format: `<signature_counter>_<data_to_be_signed>_<last_signature_base64>`. For the first signature (counter == 0), `base64(device.id)` is used instead of the last signature.

### List / Get Devices

```bash
curl localhost:8080/api/v1/devices              # list all
curl localhost:8080/api/v1/devices/{id}         # get by ID
```

## Architecture

```
cmd/server/       Entry point, dependency wiring, graceful shutdown
domain/           Core types, interfaces, business logic (zero external deps)
crypto/           RSA and ECDSA signer implementations (Strategy pattern)
service/          Orchestration layer with per-device mutex concurrency control
persistence/      In-memory + PostgreSQL repository implementations
api/              HTTP handlers, DTOs, middleware, routing (stdlib only)
```

### Key Design Decisions

**Per-device mutex in the service layer.** The `signature_counter` must be strictly monotonically increasing without gaps. A global lock would serialize all signing across unrelated devices. A repository-level lock would create a TOCTOU gap between read and update. The per-device mutex in the service layer locks the entire read-sign-increment-save cycle for one device while allowing other devices to operate concurrently.

**Deep copies in the in-memory repository.** `InMemoryDeviceRepository` stores and returns clones of `SignatureDevice` structs. This prevents callers from mutating shared state through pointer aliasing — the same guarantee a SQL database provides naturally. This makes the in-memory implementation a true drop-in for a future database backend.

**Two repository implementations.** The `domain.DeviceRepository` interface has both `InMemoryDeviceRepository` and `PostgresDeviceRepository` implementations. Swapping backends is a one-line change in `main.go`. The PostgreSQL implementation is tested via **testcontainers-go** — real PostgreSQL 16 in Docker, no mocks.

**Stdlib-only routing.** Go 1.22+ `http.ServeMux` supports method-based pattern matching (`"POST /api/v1/devices/{id}/sign"`), eliminating the need for external routers like chi or gorilla/mux.

**`SecuredData()` on the domain entity.** The signed data format is domain logic that depends only on the device's own state, so it lives as a method on `SignatureDevice`, not scattered across service methods.

**No private keys in API responses.** The `DeviceResponse` DTO deliberately excludes `private_key`. Only the `public_key` (base64-encoded DER) is exposed via the API.

**Multi-stage Docker build.** The final image is ~25 MB (Alpine + binary), runs as `nobody:nobody`, and includes no compiler, source code, or build tools.

### Extensibility

To add a new signing algorithm (e.g., Ed25519):

1. Implement the `domain.Signer` interface in `crypto/`
2. Add a `GenerateXxxKeyPair()` function
3. Register in `crypto/signer.go` (signers map + `GenerateKeyPair` switch)
4. Add the algorithm constant to `domain/device.go`

No changes to the service layer, API handlers, or persistence layer.

## Testing

The test suite includes **62 tests** across 4 packages:

| Package | Tests | What's Covered |
|---------|-------|----------------|
| `crypto` | 14 | Key generation, sign + verify roundtrip with public key, RSA determinism, ECDSA non-determinism |
| `persistence` | 16 | In-memory CRUD + deep-copy isolation; PostgreSQL CRUD via testcontainers-go (real PostgreSQL 16) |
| `service` | 13 | Signature chain integrity, base case, counter monotonicity, **100-goroutine concurrent signing** |
| `api` | 19 | Full HTTP request/response cycle, all status codes (201/200/400/404/409), end-to-end 5-signature chain |

```bash
make test               # unit tests — fast, no Docker (~2s)
make test-race          # unit tests + race detector (~3s)
make test-integration   # PostgreSQL tests via testcontainers-go (~14s, requires Docker)
```

`make test` and `make test-race` use `-short` flag to skip PostgreSQL integration tests. `make test-integration` runs only the PostgreSQL tests with the race detector enabled.

## Docker

```bash
make docker-build       # build image
make docker-run         # build and run on :8080
```

The Dockerfile uses a multi-stage build:
- **Build stage**: `golang:1.25-alpine` — compiles a statically linked binary with `-ldflags="-s -w"`
- **Runtime stage**: `alpine:3.21` — only the binary, ca-certificates, and tzdata
- **Final image size**: ~25 MB
- **Runs as**: `nobody:nobody` (non-root)

## AI Tools Disclosure

This project was developed with assistance from **Claude Code** (Anthropic's CLI tool, Claude Opus 4.6 model). Claude Code was used for:

- Architecture design and implementation planning
- Code generation across all packages
- Test case design and implementation
- README documentation

All design decisions (per-device mutex strategy, deep-copy isolation, stdlib routing, Repository pattern with two implementations, key serialization format) were reviewed, understood, and can be reasoned about in detail during the interview.
