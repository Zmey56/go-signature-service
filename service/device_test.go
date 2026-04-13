package service

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/alekstut/signing-service-challenge/domain"
	"github.com/alekstut/signing-service-challenge/persistence"
	"github.com/google/uuid"
)

func newService() *DeviceService {
	return NewDeviceService(persistence.NewInMemoryRepository())
}

func TestCreateDevice_ECC(t *testing.T) {
	svc := newService()
	id := uuid.NewString()

	device, err := svc.CreateDevice(id, domain.AlgorithmECC, "my device")
	if err != nil {
		t.Fatalf("CreateDevice() error: %v", err)
	}
	if device.ID != id {
		t.Fatalf("expected ID %s, got %s", id, device.ID)
	}
	if device.Algorithm != domain.AlgorithmECC {
		t.Fatalf("expected ECC, got %s", device.Algorithm)
	}
	if device.Label != "my device" {
		t.Fatalf("expected label 'my device', got %s", device.Label)
	}
	if device.SignatureCounter != 0 {
		t.Fatalf("expected counter 0, got %d", device.SignatureCounter)
	}
	if len(device.PrivateKey) == 0 || len(device.PublicKey) == 0 {
		t.Fatal("keys should not be empty")
	}
}

func TestCreateDevice_RSA(t *testing.T) {
	svc := newService()
	device, err := svc.CreateDevice(uuid.NewString(), domain.AlgorithmRSA, "rsa device")
	if err != nil {
		t.Fatalf("CreateDevice() error: %v", err)
	}
	if device.Algorithm != domain.AlgorithmRSA {
		t.Fatalf("expected RSA, got %s", device.Algorithm)
	}
}

func TestCreateDevice_DuplicateID(t *testing.T) {
	svc := newService()
	id := uuid.NewString()

	if _, err := svc.CreateDevice(id, domain.AlgorithmECC, ""); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.CreateDevice(id, domain.AlgorithmECC, ""); err != domain.ErrDeviceAlreadyExists {
		t.Fatalf("expected ErrDeviceAlreadyExists, got %v", err)
	}
}

func TestCreateDevice_InvalidAlgorithm(t *testing.T) {
	svc := newService()
	_, err := svc.CreateDevice(uuid.NewString(), "EdDSA", "")
	if err != domain.ErrUnsupportedAlgorithm {
		t.Fatalf("expected ErrUnsupportedAlgorithm, got %v", err)
	}
}

func TestSignTransaction_Success(t *testing.T) {
	svc := newService()
	id := uuid.NewString()
	svc.CreateDevice(id, domain.AlgorithmECC, "")

	sig, signedData, err := svc.SignTransaction(id, "tx-data-1")
	if err != nil {
		t.Fatalf("SignTransaction() error: %v", err)
	}
	if sig == "" {
		t.Fatal("signature is empty")
	}
	if signedData == "" {
		t.Fatal("signed_data is empty")
	}

	// Verify counter was incremented
	device, _ := svc.GetDevice(id)
	if device.SignatureCounter != 1 {
		t.Fatalf("expected counter 1, got %d", device.SignatureCounter)
	}
}

func TestSignTransaction_BaseCase(t *testing.T) {
	svc := newService()
	id := uuid.NewString()
	svc.CreateDevice(id, domain.AlgorithmECC, "")

	_, signedData, err := svc.SignTransaction(id, "hello")
	if err != nil {
		t.Fatalf("SignTransaction() error: %v", err)
	}

	expectedLastSig := base64.StdEncoding.EncodeToString([]byte(id))
	expected := fmt.Sprintf("0_hello_%s", expectedLastSig)
	if signedData != expected {
		t.Fatalf("signed_data mismatch:\n  got:  %s\n  want: %s", signedData, expected)
	}
}

func TestSignTransaction_ChainIntegrity(t *testing.T) {
	svc := newService()
	id := uuid.NewString()
	svc.CreateDevice(id, domain.AlgorithmECC, "")

	var lastSig string
	for i := range 3 {
		sig, signedData, err := svc.SignTransaction(id, fmt.Sprintf("tx-%d", i))
		if err != nil {
			t.Fatalf("sign %d: %v", i, err)
		}

		parts := strings.SplitN(signedData, "_", 3)
		if len(parts) != 3 {
			t.Fatalf("signed_data format invalid: %s", signedData)
		}

		counter := parts[0]
		if counter != fmt.Sprintf("%d", i) {
			t.Fatalf("expected counter %d, got %s", i, counter)
		}

		if i == 0 {
			expectedLastSig := base64.StdEncoding.EncodeToString([]byte(id))
			if parts[2] != expectedLastSig {
				t.Fatalf("base case: expected base64(id), got %s", parts[2])
			}
		} else {
			if parts[2] != lastSig {
				t.Fatalf("chain broken at %d: expected %s, got %s", i, lastSig, parts[2])
			}
		}

		lastSig = sig
	}
}

func TestSignTransaction_DeviceNotFound(t *testing.T) {
	svc := newService()
	_, _, err := svc.SignTransaction("nonexistent", "data")
	if err != domain.ErrDeviceNotFound {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestSignTransaction_CounterMonotonicity(t *testing.T) {
	svc := newService()
	id := uuid.NewString()
	svc.CreateDevice(id, domain.AlgorithmECC, "")

	n := 20
	for i := range n {
		_, _, err := svc.SignTransaction(id, fmt.Sprintf("tx-%d", i))
		if err != nil {
			t.Fatalf("sign %d: %v", i, err)
		}
	}

	device, _ := svc.GetDevice(id)
	if device.SignatureCounter != n {
		t.Fatalf("expected counter %d, got %d", n, device.SignatureCounter)
	}
}

func TestSignTransaction_ConcurrentSameDevice(t *testing.T) {
	svc := newService()
	id := uuid.NewString()
	svc.CreateDevice(id, domain.AlgorithmECC, "")

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, goroutines)

	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			_, _, err := svc.SignTransaction(id, fmt.Sprintf("tx-%d", i))
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent sign error: %v", err)
	}

	device, _ := svc.GetDevice(id)
	if device.SignatureCounter != goroutines {
		t.Fatalf("expected counter %d, got %d", goroutines, device.SignatureCounter)
	}
}

func TestSignTransaction_ConcurrentDifferentDevices(t *testing.T) {
	svc := newService()
	const devices = 10
	const signsPerDevice = 10

	ids := make([]string, devices)
	for i := range devices {
		ids[i] = uuid.NewString()
		svc.CreateDevice(ids[i], domain.AlgorithmECC, "")
	}

	var wg sync.WaitGroup
	wg.Add(devices * signsPerDevice)
	errs := make(chan error, devices*signsPerDevice)

	for _, id := range ids {
		for j := range signsPerDevice {
			go func(id string, j int) {
				defer wg.Done()
				_, _, err := svc.SignTransaction(id, fmt.Sprintf("tx-%d", j))
				if err != nil {
					errs <- err
				}
			}(id, j)
		}
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent sign error: %v", err)
	}

	for _, id := range ids {
		device, _ := svc.GetDevice(id)
		if device.SignatureCounter != signsPerDevice {
			t.Fatalf("device %s: expected counter %d, got %d", id, signsPerDevice, device.SignatureCounter)
		}
	}
}

func TestListDevices(t *testing.T) {
	svc := newService()
	for i := range 3 {
		svc.CreateDevice(uuid.NewString(), domain.AlgorithmECC, fmt.Sprintf("dev-%d", i))
	}

	devices, err := svc.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices() error: %v", err)
	}
	if len(devices) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(devices))
	}
}

func TestGetDevice_NotFound(t *testing.T) {
	svc := newService()
	_, err := svc.GetDevice("nonexistent")
	if err != domain.ErrDeviceNotFound {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}
