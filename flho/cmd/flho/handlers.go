package main

import (
	"encoding/json"
	"net/http"
)

type envelope map[any]any

func writeResponse(w http.ResponseWriter, statusCode int, data envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)

}

func (a *application) healthcheck(w http.ResponseWriter, r *http.Request) {
	writeResponse(w, http.StatusOK, envelope{
		"status": "available",
	})
}
