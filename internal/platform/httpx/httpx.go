// Package httpx holds the HTTP plumbing shared by the API entry point:
// server construction with explicit timeouts and JSON response writing.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
)

// NewServer builds an http.Server with explicit timeouts.
//
// The zero value of http.Server has no timeout at all, which lets a slow client
// hold a connection open indefinitely.
func NewServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

// WriteJSON serialises payload with the given status code.
func WriteJSON(w http.ResponseWriter, logger *slog.Logger, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		// The status line is already on the wire, so the only thing left is to
		// record that the body could not be completed.
		logger.Error("cannot encode the response body", slog.Any("error", err))
	}
}
