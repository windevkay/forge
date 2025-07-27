// Package service provides the core workflow execution engine for the flho system.
// This package implements a distributed workflow orchestration service that manages
// the execution of multi-step workflows with automatic retry mechanisms.
//
// The service is designed to handle long-running workflows where each step may
// fail and require retries after specified intervals. It provides persistent
// state management and supports workflow cancellation, completion tracking,
// and step progression.
//
// Key Features:
//   - Asynchronous workflow execution with goroutine-based step processing
//   - Automatic retry mechanisms with configurable intervals
//   - Thread-safe workflow state management using sync.Map
//   - Persistent state storage via the genie key-value store
//   - HTTP-based retry notifications to external services
//   - Context-based cancellation and timeout support
//   - Workflow run tracking with start/end timestamps
//
// Architecture:
//
// The WorkflowService orchestrates workflow execution by:
//  1. Initiating workflows and generating unique run IDs
//  2. Processing individual workflow steps in separate goroutines
//  3. Managing retry timers for failed steps
//  4. Persisting workflow state for resumption after failures
//  5. Sending HTTP notifications to retry URLs when steps need attention
//  6. Tracking active runs and their cancellation functions
//
// Workflow Lifecycle:
//
//	┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
//	│  InitiateWorkflow│───▶│  processStep     │───▶│ CompleteWorkflow│
//	└─────────────────┘    └──────────────────┘    └─────────────────┘
//	                              │                         ▲
//	                              ▼                         │
//	                       ┌──────────────────┐             │
//	                       │  UpdateWorkflow  │─────────────┘
//	                       └──────────────────┘
//
// Usage Example (direct vs using REST endpoints):
//
//	config := workflow.NewConfigStore()
//	store, _ := genie.NewStore()
//	wg := &sync.WaitGroup{}
//
//	service := NewWorkflowService(config, store, wg)
//
//	// Start a workflow
//	runID, err := service.InitiateWorkflow(context.Background(), "user_onboarding")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Progress to next step
//	err = service.UpdateWorkflow(context.Background(), runID)
//
//	// Mark as complete
//	err = service.CompleteWorkflow(runID)
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/windevkay/forge/flho/internal/workflow"
	"github.com/windevkay/forge/genie"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type UUIDProvider interface {
	NewString() string
}

type TimeProvider interface {
	Now() time.Time
}

// Production implementations

type DefaultUUIDProvider struct{}

func (p *DefaultUUIDProvider) NewString() string {
	return uuid.NewString()
}

type DefaultTimeProvider struct{}

func (p *DefaultTimeProvider) Now() time.Time {
	return time.Now()
}

// NewWorkflowService creates a new WorkflowService with default production implementations
// for HTTP client, UUID provider, and time provider.
func NewWorkflowService(cfg *workflow.ConfigStore, store *genie.Store, wg *sync.WaitGroup, logger *slog.Logger) *WorkflowService {
	return NewService(
		cfg,
		store,
		wg,
		logger,
		&http.Client{},
		&DefaultUUIDProvider{},
		&DefaultTimeProvider{},
	)
}

type WorkflowService struct {
	config       *workflow.ConfigStore
	httpClient   HTTPClient
	uuidProvider UUIDProvider
	timeProvider TimeProvider
	logger       *slog.Logger
	runs         sync.Map
	store        *genie.Store
	wg           *sync.WaitGroup
}

// NewService creates a new instance of WorkflowService with the provided configuration,
// store, and wait group for managing workflow executions.
func NewService(cfg *workflow.ConfigStore, store *genie.Store, wg *sync.WaitGroup, logger *slog.Logger, httpClient HTTPClient, uuidProvider UUIDProvider, timeProvider TimeProvider) *WorkflowService {
	return &WorkflowService{
		config:       cfg,
		httpClient:   httpClient,
		uuidProvider: uuidProvider,
		timeProvider: timeProvider,
		logger:       logger,
		store:        store,
		wg:           wg,
	}
}

type Run struct {
	currStep     int
	failed       bool
	workflowName string
	retryCancel  context.CancelFunc
	start, end   *time.Time
}

// InitiateWorkflow starts a new workflow instance with the given name, returning a unique run ID.
// It initiates the first step of the workflow in a separate goroutine.
func (w *WorkflowService) InitiateWorkflow(ctx context.Context, name string) string {
	index := 0 // starting a new workflow so defaulting to first step

	runID := w.uuidProvider.NewString()
	runCtx, cancel := context.WithCancel(ctx)
	runstart := w.timeProvider.Now()
	run := &Run{
		currStep:     index,
		workflowName: name,
		retryCancel:  cancel,
		start:        &runstart,
	}

	w.runs.Store(runID, run)

	w.wg.Add(1)
	go w.processStep(runCtx, index, runID, name)

	return runID
}

// UpdateWorkflow progresses the specified workflow by one step.
// It retrieves the current step index and processes the next step.
func (w *WorkflowService) UpdateWorkflow(ctx context.Context, runID string) error {
	currentIdx, existing := w.store.Get(runID)
	if !existing {
		return fmt.Errorf("no data found for run ID: %s", runID)
	}

	run, err := w.cancelRetryCountdown(runID)
	if err != nil {
		return err
	}

	// getting to this point means we now need to process the next step
	cIdx, _ := strconv.Atoi(currentIdx)
	// create a fresh run context and cancel func
	// also update the current runs step
	runCtx, cancel := context.WithCancel(ctx)
	run.retryCancel = cancel

	step := cIdx + 1
	run.currStep = step

	w.runs.Store(runID, run)

	w.wg.Add(1)
	go w.processStep(runCtx, step, runID, run.workflowName)

	return nil
}

// CompleteWorkflow finalizes the specified workflow run.
// It cancels any pending retries and marks the workflow end time.
func (w *WorkflowService) CompleteWorkflow(runID string) error {
	run, err := w.cancelRetryCountdown(runID)
	if err != nil {
		return err
	}

	runEnd := w.timeProvider.Now()
	run.end = &runEnd

	w.runs.Store(runID, run)

	return nil
}

// processStep executes a single step in the workflow, managing retries and HTTP notifications.
// It stops when the context is done or after a successful HTTP POST request.
func (w *WorkflowService) processStep(ctx context.Context, index int, runID, name string) {
	defer w.wg.Done()

	step := fmt.Sprintf("step%v", index)

	workflow := w.config.GetWorkflows()[name]
	if workflow == nil || len(workflow) <= index {
		w.logger.Error("encountered a step with no config - workflow not found or invalid index")
		return
	}

	if _, ok := workflow[index][step]; !ok {
		w.logger.Error("encountered a step with no config")
		return
	}

	w.store.Set(runID, strconv.Itoa(index))

	stepData := workflow[index][step]

	ticker := time.NewTicker(stepData.RetryAfter)

	for {
		select {
		case <-ticker.C:
			// curate the data the client can utilize for retries within their app
			// ideally this information can be used as a key to fetch the appropriate
			// function that needs to be called/retried + its arguments
			retryData := struct {
				WorkflowName  string `json:"workflow_name"`
				WorkflowStep  string `json:"workflow_step"`
				WorkflowRunID string `json:"workflow_run_id"`
			}{
				WorkflowName:  name,
				WorkflowStep:  step,
				WorkflowRunID: runID,
			}

			jsonData, _ := json.Marshal(retryData)

			// Create HTTP request with context
			req, err := http.NewRequestWithContext(ctx, "POST", stepData.RetryURL, bytes.NewBuffer(jsonData))
			if err != nil {
				w.logger.Error("failed to create HTTP request")
				return
			}
			req.Header.Set("Content-Type", "application/json")

			client := w.httpClient
			res, err := client.Do(req)
			if err != nil {
				w.logger.Error("POST to retryURL unsuccessful")
				return
			}
			_ = res.Body.Close()
			// mark run as failed
			w.markRunAsFailed(runID)
			return
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}

// cancelRetryCountdown cancels any pending retries for the specified run ID.
// It retrieves and returns the run information.
func (w *WorkflowService) cancelRetryCountdown(runID string) (*Run, error) {
	r, ok := w.runs.Load(runID)
	if !ok {
		return nil, errors.New("run information missing. Did a previous step fail?")
	}
	run := r.(*Run)

	run.retryCancel()

	return run, nil
}

// help to mark a failed run and update the end timestamp
func (w *WorkflowService) markRunAsFailed(runID string) {
	r, _ := w.runs.Load(runID)
	run := r.(*Run)
	run.failed = true
	runEnd := w.timeProvider.Now()
	run.end = &runEnd

	w.runs.Store(runID, run)
}
