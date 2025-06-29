package main

import (
	"encoding/json"
	"net/http"
)

type envelope map[string]any

type InitiateWorkflowRequest struct {
	Name string `json:"name"`
}

type UpdateWorkflowRequest struct {
	RunID string `json:"run_id"`
}

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

func (app *application) initiateWorkflow(w http.ResponseWriter, r *http.Request) {
	var request InitiateWorkflowRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		app.writeResponse(w, http.StatusBadRequest, envelope{
			"error": "Invalid JSON",
		})
		return
	}

	runID := app.service.InitiateWorkflow(r.Context(), request.Name)
	app.writeResponse(w, http.StatusCreated, envelope{
		"run_id": runID,
	})
}

func (app *application) updateWorkflow(w http.ResponseWriter, r *http.Request) {
	var request UpdateWorkflowRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		app.writeResponse(w, http.StatusBadRequest, envelope{
			"error": "Invalid JSON",
		})
		return
	}

	err := app.service.UpdateWorkflow(r.Context(), request.RunID)
	if err != nil {
		app.writeResponse(w, http.StatusBadRequest, envelope{
			"error": err.Error(),
		})
		return
	}

	app.writeResponse(w, http.StatusBadRequest, envelope{
		"success": "run updated",
	})
}

func (app *application) completeWorkflow(w http.ResponseWriter, r *http.Request) {
	var request UpdateWorkflowRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		app.writeResponse(w, http.StatusBadRequest, envelope{
			"error": "Invalid JSON",
		})
		return
	}

	err := app.service.CompleteWorkflow(request.RunID)
	if err != nil {
		app.writeResponse(w, http.StatusBadRequest, envelope{
			"error": err.Error(),
		})
		return
	}

	app.writeResponse(w, http.StatusBadRequest, envelope{
		"success": "run completed",
	})
}
