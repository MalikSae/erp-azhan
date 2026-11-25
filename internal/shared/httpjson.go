package shared

import (
	"encoding/json"
	"net/http"
)

// WriteJSON menulis respons JSON dengan status yang diberikan.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// WriteError menulis respons galat standar {"error": "..."}.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}
