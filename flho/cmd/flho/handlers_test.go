package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"log/slog"
	"os"

	"github.com/windevkay/forge/flho/internal/service"
	"github.com/windevkay/forge/flho/internal/workflow"
	"github.com/windevkay/forge/genie/v2"
)

func TestListRunsHandler(t *testing.T) {
	// Setup test application
	config := &workflow.ConfigStore{}
	store, err := genie.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	
	wg := &sync.WaitGroup{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	app := &application{
		service: service.NewWorkflowService(config, store, wg, logger),
		logger:  logger,
	}

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/runs", nil)
	w := httptest.NewRecorder()

	// Execute handler
	app.listRuns(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected content type 'text/html; charset=utf-8', got '%s'", contentType)
	}

	// Check that response contains some basic HTML
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}
}

func TestListRunsHandlerWithFilters(t *testing.T) {
	// Setup test application
	config := &workflow.ConfigStore{}
	store, err := genie.NewStore()
	if err != nil {
		t.Fatal(err)
	}
	
	wg := &sync.WaitGroup{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	app := &application{
		service: service.NewWorkflowService(config, store, wg, logger),
		logger:  logger,
	}

	// Create test request with query parameters
	req := httptest.NewRequest(http.MethodGet, "/runs?status=ongoing&workflow=test&page=1", nil)
	w := httptest.NewRecorder()

	// Execute handler
	app.listRuns(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
