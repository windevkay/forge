package main

import (
	"encoding/json"
	"net/http"
)

type envelope map[string]any

func (app *application) writeResponse(w http.ResponseWriter, statusCode int, data envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		app.logger.Error(err.Error())
	}
}

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {
	app.writeResponse(w, http.StatusOK, envelope{
		"status": "available",
	})
}
