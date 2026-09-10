package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// errorPayload is the body returned by ErrorResponse.
type errorPayload struct {
	Error string `json:"error"`
}

// OKResponse writes data as a JSON body with a 200 OK status.
func OKResponse(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, data)
}

// CreatedResponse writes data as a JSON body with a 201 Created status.
func CreatedResponse(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, data)
}

// ErrorResponse writes message as a JSON body with the given status code.
func ErrorResponse(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorPayload{Error: message})
}

// writeJSON encodes data as the JSON body of a response. Headers must be set
// before the status code is written, otherwise they are silently dropped.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// The status code is already on the wire, so the client cannot be told
		// about this anymore. Log it and move on.
		log.Printf("api: encoding response failed: %s", err)
	}
}
