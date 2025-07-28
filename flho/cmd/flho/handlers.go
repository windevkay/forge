package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/windevkay/forge/flho/internal/service"
)

type envelope map[string]any

// InitiateWorkflowRequest represents the request body for initiating a workflow
type InitiateWorkflowRequest struct {
	Name string `json:"name"`
}

// UpdateWorkflowRequest represents the request body for updating a workflow
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

func (app *application) healthcheck(w http.ResponseWriter, _ *http.Request) {
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

func (app *application) listRuns(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	defaultInt := 20

	// Extract filters from query parameters
	status := query.Get("status")
	workflowName := query.Get("workflow")
	page := 1
	pageSize := 20

	if p := query.Get("page"); p != "" {
		page = parseInt(p, 1)
	}
	if ps := query.Get("pageSize"); ps != "" {
		pageSize = parseInt(ps, defaultInt)
	}

	// Create filter based on query params
	filter := service.RunsFilter{
		Status:       status,
		WorkflowName: workflowName,
		Page:         page,
		PageSize:     pageSize,
	}

	// Retrieve runs based on the filter
	runsResponse := app.service.GetRuns(filter)

	// Render template with Bootstrap styling
	app.renderHTML(w, "runs.page.html", runsResponse)
}

func parseInt(val string, defaultInt int) int {
	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultInt
	}
	return result
}

func (app *application) renderHTML(w http.ResponseWriter, _ string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, err := template.New("runs.html").Funcs(template.FuncMap{
		"formatTime": func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.Format("2006-01-02 15:04:05")
		},
		"formatDuration": func(d *time.Duration) string {
			if d == nil {
				return "-"
			}
			return d.String()
		},
		"statusBadge": func(status service.RunStatus) string {
			switch status {
			case service.RunStatusOngoing:
				return "bg-primary"
			case service.RunStatusCompleted:
				return "bg-success"
			case service.RunStatusFailed:
				return "bg-danger"
			default:
				return "bg-secondary"
			}
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"iterate": func(count int) []int {
			var items []int
			for i := 0; i < count; i++ {
				items = append(items, i)
			}
			return items
		},
		"eq": func(a, b interface{}) bool { return a == b },
		"gt": func(a, b int) bool { return a > b },
		"lt": func(a, b int) bool { return a < b },
	}).ParseFiles("web/templates/runs.html")

	if err != nil {
		app.logger.Error("Template parse error: " + err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		app.logger.Error("Template execution error: " + err.Error())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
