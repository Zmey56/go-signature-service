package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alekstut/signing-service-challenge/persistence"
	"github.com/alekstut/signing-service-challenge/service"
	"github.com/google/uuid"
)

func setupServer() http.Handler {
	repo := persistence.NewInMemoryRepository()
	svc := service.NewDeviceService(repo)
	handler := NewDeviceHandler(svc)
	return NewServer(handler)
}

func createDevice(t *testing.T, srv http.Handler, id, algorithm, label string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"id":"%s","algorithm":"%s","label":"%s"}`, id, algorithm, label)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func signTransaction(t *testing.T, srv http.Handler, deviceID, data string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"data":"%s"}`, data)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+deviceID+"/sign", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func TestCreateDevice_HTTP(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()

	w := createDevice(t, srv, id, "ECC", "test device")
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp DeviceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != id {
		t.Fatalf("expected ID %s, got %s", id, resp.ID)
	}
	if resp.Algorithm != "ECC" {
		t.Fatalf("expected ECC, got %s", resp.Algorithm)
	}
	if resp.PublicKey == "" {
		t.Fatal("public_key should not be empty")
	}
}

func TestCreateDevice_HTTP_RSA(t *testing.T) {
	srv := setupServer()
	w := createDevice(t, srv, uuid.NewString(), "RSA", "rsa device")
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateDevice_HTTP_InvalidJSON(t *testing.T) {
	srv := setupServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader("{invalid"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateDevice_HTTP_MissingAlgorithm(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()
	body := fmt.Sprintf(`{"id":"%s"}`, id)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateDevice_HTTP_InvalidAlgorithm(t *testing.T) {
	srv := setupServer()
	w := createDevice(t, srv, uuid.NewString(), "EdDSA", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateDevice_HTTP_DuplicateID(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()
	createDevice(t, srv, id, "ECC", "")
	w := createDevice(t, srv, id, "ECC", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestCreateDevice_HTTP_InvalidUUID(t *testing.T) {
	srv := setupServer()
	w := createDevice(t, srv, "not-a-uuid", "ECC", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetDevice_HTTP(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()
	createDevice(t, srv, id, "ECC", "my device")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+id, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp DeviceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ID != id || resp.Label != "my device" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetDevice_HTTP_NotFound(t *testing.T) {
	srv := setupServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+uuid.NewString(), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestListDevices_HTTP(t *testing.T) {
	srv := setupServer()
	for range 3 {
		createDevice(t, srv, uuid.NewString(), "ECC", "")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []DeviceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(resp))
	}
}

func TestListDevices_HTTP_Empty(t *testing.T) {
	srv := setupServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []DeviceResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Fatalf("expected 0 devices, got %d", len(resp))
	}
}

func TestSignTransaction_HTTP(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()
	createDevice(t, srv, id, "ECC", "")

	w := signTransaction(t, srv, id, "hello world")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SignatureResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Signature == "" {
		t.Fatal("signature should not be empty")
	}
	if resp.SignedData == "" {
		t.Fatal("signed_data should not be empty")
	}
}

func TestSignTransaction_HTTP_EmptyData(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()
	createDevice(t, srv, id, "ECC", "")

	body := `{"data":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id+"/sign", strings.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSignTransaction_HTTP_DeviceNotFound(t *testing.T) {
	srv := setupServer()
	w := signTransaction(t, srv, uuid.NewString(), "data")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSignTransaction_HTTP_InvalidJSON(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()
	createDevice(t, srv, id, "ECC", "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/"+id+"/sign", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestFullSignatureChain_HTTP(t *testing.T) {
	srv := setupServer()
	id := uuid.NewString()
	createDevice(t, srv, id, "ECC", "chain test")

	var lastSig string
	for i := range 5 {
		w := signTransaction(t, srv, id, fmt.Sprintf("tx-%d", i))
		if w.Code != http.StatusOK {
			t.Fatalf("sign %d: expected 200, got %d", i, w.Code)
		}

		var resp SignatureResponse
		json.NewDecoder(w.Body).Decode(&resp)

		parts := strings.SplitN(resp.SignedData, "_", 3)
		if len(parts) != 3 {
			t.Fatalf("signed_data format invalid: %s", resp.SignedData)
		}
		if parts[0] != fmt.Sprintf("%d", i) {
			t.Fatalf("expected counter %d, got %s", i, parts[0])
		}
		if i > 0 && parts[2] != lastSig {
			t.Fatalf("chain broken at %d", i)
		}

		lastSig = resp.Signature
	}

	// Verify counter via GET
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/"+id, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var device DeviceResponse
	json.NewDecoder(w.Body).Decode(&device)
	if device.SignatureCounter != 5 {
		t.Fatalf("expected counter 5, got %d", device.SignatureCounter)
	}
}

func TestHealth_HTTP(t *testing.T) {
	srv := setupServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", resp["status"])
	}
}

func TestContentTypeHeader(t *testing.T) {
	srv := setupServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}
}

func TestRequestID_Generated(t *testing.T) {
	srv := setupServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	id := w.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
	// Should be a valid UUID
	if len(id) != 36 {
		t.Fatalf("expected UUID format, got %s", id)
	}
}

func TestRequestID_Propagated(t *testing.T) {
	srv := setupServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "my-trace-id-123")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	id := w.Header().Get("X-Request-ID")
	if id != "my-trace-id-123" {
		t.Fatalf("expected propagated request ID 'my-trace-id-123', got %s", id)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	srv := RecoveryMiddleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

