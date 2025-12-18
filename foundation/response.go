package foundation

import (
	"encoding/json"
	"net/http"
)

// NoResponse tells the Respond function to not respond to the request. In these
// cases the app layer code has already done so.
type NoResponse struct{}

// Respond sends a response to the client.
func Respond(w http.ResponseWriter, code int, v any) {
	if _, ok := v.(NoResponse); ok {
		w.WriteHeader(code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
