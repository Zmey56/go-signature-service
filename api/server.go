package api

import "net/http"

// NewServer creates an http.Handler with all routes and middleware wired up.
func NewServer(handler *DeviceHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/devices", handler.CreateDevice)
	mux.HandleFunc("GET /api/v1/devices", handler.ListDevices)
	mux.HandleFunc("GET /api/v1/devices/{id}", handler.GetDevice)
	mux.HandleFunc("POST /api/v1/devices/{id}/sign", handler.SignTransaction)
	mux.HandleFunc("GET /health", handler.Health)

	// Apply middleware: recovery -> request-id -> logging -> content-type -> router
	return RecoveryMiddleware(RequestIDMiddleware(LoggingMiddleware(ContentTypeMiddleware(mux))))
}
