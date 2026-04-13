package service

import (
	"encoding/base64"
	"sync"
	"time"

	"github.com/alekstut/signing-service-challenge/crypto"
	"github.com/alekstut/signing-service-challenge/domain"
)

// DeviceService orchestrates signature device operations.
type DeviceService struct {
	repo     domain.DeviceRepository
	mu       sync.Mutex             // protects deviceMu map
	deviceMu map[string]*sync.Mutex // per-device mutex for sign operations
}

// NewDeviceService creates a new DeviceService.
func NewDeviceService(repo domain.DeviceRepository) *DeviceService {
	return &DeviceService{
		repo:     repo,
		deviceMu: make(map[string]*sync.Mutex),
	}
}

// getDeviceMutex returns the per-device mutex, creating one if needed.
func (s *DeviceService) getDeviceMutex(deviceID string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.deviceMu[deviceID]
	if !ok {
		m = &sync.Mutex{}
		s.deviceMu[deviceID] = m
	}
	return m
}

// CreateDevice creates a new signature device with a generated key pair.
func (s *DeviceService) CreateDevice(id string, algorithm domain.Algorithm, label string) (*domain.SignatureDevice, error) {
	deviceMu := s.getDeviceMutex(id)
	deviceMu.Lock()
	defer deviceMu.Unlock()

	privKey, pubKey, err := crypto.GenerateKeyPair(algorithm)
	if err != nil {
		return nil, err
	}

	device := &domain.SignatureDevice{
		ID:         id,
		Algorithm:  algorithm,
		Label:      label,
		PrivateKey: privKey,
		PublicKey:  pubKey,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Save(device); err != nil {
		return nil, err
	}

	return device, nil
}

// SignTransaction atomically signs data and increments the counter.
func (s *DeviceService) SignTransaction(deviceID, data string) (signature string, signedData string, err error) {
	deviceMu := s.getDeviceMutex(deviceID)
	deviceMu.Lock()
	defer deviceMu.Unlock()

	device, err := s.repo.FindByID(deviceID)
	if err != nil {
		return "", "", err
	}

	signedData = device.SecuredData(data)

	signer, err := crypto.NewSigner(device.Algorithm)
	if err != nil {
		return "", "", err
	}

	sigBytes, err := signer.Sign([]byte(signedData), device.PrivateKey)
	if err != nil {
		return "", "", err
	}

	device.SignatureCounter++
	device.LastSignature = base64.StdEncoding.EncodeToString(sigBytes)

	if err := s.repo.Save(device); err != nil {
		return "", "", err
	}

	return device.LastSignature, signedData, nil
}

// GetDevice retrieves a single device by ID.
func (s *DeviceService) GetDevice(id string) (*domain.SignatureDevice, error) {
	return s.repo.FindByID(id)
}

// ListDevices returns all devices.
func (s *DeviceService) ListDevices() ([]*domain.SignatureDevice, error) {
	return s.repo.FindAll()
}
