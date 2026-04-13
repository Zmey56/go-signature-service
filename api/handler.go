package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alekstut/signing-service-challenge/domain"
	"github.com/alekstut/signing-service-challenge/service"
)

// DeviceHandler handles HTTP requests for signature device operations.
type DeviceHandler struct {
	service *service.DeviceService
}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler(svc *service.DeviceService) *DeviceHandler {
	return &DeviceHandler{service: svc}
}

// CreateDevice handles POST /api/v1/devices
func (h *DeviceHandler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var req CreateDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	alg, err := domain.ParseAlgorithm(req.Algorithm)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	device, err := h.service.CreateDevice(req.ID, alg, req.Label)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, NewDeviceResponse(device))
}

// ListDevices handles GET /api/v1/devices
func (h *DeviceHandler) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := h.service.ListDevices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list devices")
		return
	}

	resp := make([]DeviceResponse, len(devices))
	for i, d := range devices {
		resp[i] = NewDeviceResponse(d)
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetDevice handles GET /api/v1/devices/{id}
func (h *DeviceHandler) GetDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	device, err := h.service.GetDevice(id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, NewDeviceResponse(device))
}

// SignTransaction handles POST /api/v1/devices/{id}/sign
func (h *DeviceHandler) SignTransaction(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req SignTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	sig, signedData, err := h.service.SignTransaction(id, req.Data)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, SignatureResponse{
		Signature:  sig,
		SignedData: signedData,
	})
}

// Health handles GET /health
func (h *DeviceHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: msg})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrDeviceNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrDeviceAlreadyExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrUnsupportedAlgorithm):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
