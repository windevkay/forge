package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/windevkay/forge/flho/internal/workflow"
	"github.com/windevkay/forge/genie"
)

type WorkflowService struct {
	config *workflow.ConfigStore
	runs   sync.Map
	store  *genie.Store
	wg     *sync.WaitGroup
}

func NewWorkflowService(cfg *workflow.ConfigStore, store *genie.Store, wg *sync.WaitGroup) *WorkflowService {
	return &WorkflowService{
		config: cfg,
		store:  store,
		wg:     wg,
	}
}

type Run struct {
	workflowName string
	retryCancel  context.CancelFunc
	start, end   *time.Time
}

func (w *WorkflowService) InitiateWorkflow(ctx context.Context, name string) (string, error) {
	index := 0 // starting a new workflow so defaulting to first step

	runID := uuid.NewString()
	runCtx, cancel := context.WithCancel(ctx)
	runstart := time.Now()
	run := &Run{
		workflowName: name,
		retryCancel:  cancel,
		start:        &runstart,
	}

	w.runs.Store(runID, run)

	w.wg.Add(1)
	go w.processStep(runCtx, index, runID, name)

	return runID, nil
}

func (w *WorkflowService) UpdateWorkflow(ctx context.Context, runID string) error {
	currentIdx, existing := w.store.Get(runID)
	if !existing {
		return fmt.Errorf("no data found for run ID: %s", runID)
	}
	// cancel the retry countdown for the existing step
	r, ok := w.runs.Load(runID)
	if !ok {
		// TODO: this is critical -
		return errors.New("missing run data")
	}

	run := r.(*Run)

	run.retryCancel()

	// getting to this point means we now need to process the next step
	cIdx, _ := strconv.Atoi(currentIdx)
	runCtx, cancel := context.WithCancel(ctx)
	run.retryCancel = cancel

	w.runs.Store(runID, run)

	w.wg.Add(1)
	go w.processStep(runCtx, cIdx+1, runID, run.workflowName)

	return nil
}

func (w *WorkflowService) CompleteWorkflow(runID string) error {
	r, _ := w.runs.Load(runID)
	run := r.(*Run)

	run.retryCancel()

	runEnd := time.Now()
	run.end = &runEnd

	w.runs.Store(runID, run)

	return nil
}

func (w *WorkflowService) processStep(ctx context.Context, index int, runID, name string) {
	defer w.wg.Done()

	step := fmt.Sprintf("step%v", index)

	workflow := w.config.GetWorkflows()[name]
	if _, ok := workflow[index][step]; !ok {
		// TODO: write error details to an err chan
		return
	}

	w.store.Set(runID, strconv.Itoa(index))

	ticker := time.NewTicker(workflow[index][step].RetryAfter)

	for {
		select {
		case <-ticker.C:
			// TODO: initiate a retry
			return
		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}
}
