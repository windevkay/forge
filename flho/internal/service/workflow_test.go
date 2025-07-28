package service

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/windevkay/forge/flho/internal/workflow"
	"github.com/windevkay/forge/genie/v2"
)

// --- Mocks ---

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

type MockUUIDProvider struct {
	mock.Mock
}

func (m *MockUUIDProvider) NewString() string {
	return m.Called().String(0)
}

type MockTimeProvider struct {
	mock.Mock
}

func (m *MockTimeProvider) Now() time.Time {
	return m.Called().Get(0).(time.Time)
}

// --- Helpers ---

func setupService(t *testing.T) (*WorkflowService, *MockUUIDProvider, *MockTimeProvider, *genie.Store) {
	mockHTTPClient := new(MockHTTPClient)
	mockUUIDProvider := new(MockUUIDProvider)
	mockTimeProvider := new(MockTimeProvider)

	// Create a real config store for testing
	config := &workflow.ConfigStore{}

	store, err := genie.NewStore()
	require.NoError(t, err)

	wg := &sync.WaitGroup{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create service with actual WorkflowService struct
	wService := &WorkflowService{
		config:       config,
		httpClient:   mockHTTPClient,
		uuidProvider: mockUUIDProvider,
		timeProvider: mockTimeProvider,
		logger:       logger,
		store:        store,
		wg:           wg,
	}

	return wService, mockUUIDProvider, mockTimeProvider, store
}

// --- Tests ---

func TestNewWorkflowService(t *testing.T) {
	store, _ := genie.NewStore()
	wg := &sync.WaitGroup{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &workflow.ConfigStore{}
	mockHTTPClient := new(MockHTTPClient)
	mockUUIDProvider := new(MockUUIDProvider)
	mockTimeProvider := new(MockTimeProvider)

	svc := NewService(config, store, wg, logger, mockHTTPClient, mockUUIDProvider, mockTimeProvider)

	require.NotNil(t, svc)
	require.Equal(t, config, svc.config)
	require.Equal(t, store, svc.store)
	require.Equal(t, wg, svc.wg)
	require.Equal(t, logger, svc.logger)
	require.Equal(t, mockHTTPClient, svc.httpClient)
	require.Equal(t, mockUUIDProvider, svc.uuidProvider)
	require.Equal(t, mockTimeProvider, svc.timeProvider)
}

func TestNewWorkflowServiceWithDefaults(t *testing.T) {
	store, _ := genie.NewStore()
	wg := &sync.WaitGroup{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &workflow.ConfigStore{}

	svc := NewWorkflowService(config, store, wg, logger)

	require.NotNil(t, svc)
	require.Equal(t, config, svc.config)
	require.Equal(t, store, svc.store)
	require.Equal(t, wg, svc.wg)
	require.Equal(t, logger, svc.logger)

	// Verify default implementations are used
	require.IsType(t, &http.Client{}, svc.httpClient)
	require.IsType(t, &DefaultUUIDProvider{}, svc.uuidProvider)
	require.IsType(t, &DefaultTimeProvider{}, svc.timeProvider)
}

func TestInitiateWorkflow(t *testing.T) {
	tests := []struct {
		name         string
		workflowName string
		expectedUUID string
		expectedTime time.Time
	}{
		{
			name:         "successful workflow initiation",
			workflowName: "test-workflow",
			expectedUUID: "123e4567-e89b-12d3-a456-426614174000",
			expectedTime: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name:         "different workflow name",
			workflowName: "another-workflow",
			expectedUUID: "987fcdeb-51d2-43a1-9876-543210987654",
			expectedTime: time.Date(2023, 2, 1, 14, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, uuidProvider, timeProvider, store := setupService(t)

			uuidProvider.On("NewString").Return(tt.expectedUUID)
			timeProvider.On("Now").Return(tt.expectedTime)

			ctx := context.Background()
			result := svc.InitiateWorkflow(ctx, tt.workflowName)

			require.Equal(t, tt.expectedUUID, result)

			// Verify run was stored
			runValue, exists := store.Get(tt.expectedUUID)
			require.True(t, exists)
			require.NotNil(t, runValue)

			run := runValue.(*Run)
			require.Equal(t, tt.workflowName, run.workflowName)
			require.Equal(t, tt.expectedTime, *run.start)
			require.Nil(t, run.end)

			uuidProvider.AssertExpectations(t)
			timeProvider.AssertExpectations(t)
		})
	}
}

func TestUpdateWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		runID       string
		setupStore  func(*genie.Store)
		expectedErr string
	}{
		{
			name:  "successful update",
			runID: "valid-run-id",
			setupStore: func(store *genie.Store) {
				_, cancel := context.WithCancel(context.Background())
				run := &Run{
					currStep:     0,
					workflowName: "test-workflow",
					retryCancel:  cancel,
				}
				store.Set("valid-run-id", run)
			},
			expectedErr: "",
		},
		{
			name:  "run ID not found in store",
			runID: "missing-run-id",
			setupStore: func(_ *genie.Store) {
				// Don't set anything
			},
			expectedErr: "no data found for run ID: missing-run-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, store := setupService(t)

			tt.setupStore(store)

			ctx := context.Background()
			err := svc.UpdateWorkflow(ctx, tt.runID)

			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Verify the step was incremented
				runValue, exists := store.Get(tt.runID)
				require.True(t, exists)
				run := runValue.(*Run)
				require.Equal(t, 1, run.currStep) // Should be incremented from 0 to 1
			}
		})
	}
}

func TestCompleteWorkflow(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		runID       string
		setupStore  func(*genie.Store)
		expectedErr string
	}{
		{
			name:  "successful completion",
			runID: "valid-run-id",
			setupStore: func(store *genie.Store) {
				_, cancel := context.WithCancel(context.Background())
				run := &Run{
					workflowName: "test-workflow",
					retryCancel:  cancel,
				}
				store.Set("valid-run-id", run)
			},
			expectedErr: "",
		},
		{
			name:  "run not found",
			runID: "missing-run-id",
			setupStore: func(_ *genie.Store) {
				// Don't set anything
			},
			expectedErr: "run information missing. Did a previous step fail?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, timeProvider, store := setupService(t)

			if tt.expectedErr == "" {
				timeProvider.On("Now").Return(fixedTime)
			}

			tt.setupStore(store)

			err := svc.CompleteWorkflow(tt.runID)

			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
			} else {
				require.NoError(t, err)

				// Verify end time was set
				runValue, exists := store.Get(tt.runID)
				require.True(t, exists)
				run := runValue.(*Run)
				require.NotNil(t, run.end)
				require.Equal(t, fixedTime, *run.end)

				timeProvider.AssertExpectations(t)
			}
		})
	}
}

func TestCancelRetryCountdown(t *testing.T) {
	tests := []struct {
		name        string
		runID       string
		setupStore  func(*genie.Store)
		expectedErr string
	}{
		{
			name:  "successful cancellation",
			runID: "valid-run-id",
			setupStore: func(store *genie.Store) {
				_, cancel := context.WithCancel(context.Background())
				run := &Run{
					workflowName: "test-workflow",
					retryCancel:  cancel,
				}
				store.Set("valid-run-id", run)
			},
			expectedErr: "",
		},
		{
			name:  "run not found",
			runID: "missing-run-id",
			setupStore: func(_ *genie.Store) {
				// Don't set anything
			},
			expectedErr: "run information missing. Did a previous step fail?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, store := setupService(t)

			tt.setupStore(store)

			run, err := svc.cancelRetryCountdown(tt.runID)

			if tt.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr)
				require.Nil(t, run)
			} else {
				require.NoError(t, err)
				require.NotNil(t, run)
				require.Equal(t, "test-workflow", run.workflowName)
			}
		})
	}
}

func TestProcessStep(t *testing.T) {
	// Note: This test focuses on the processStep function's main behavior
	// For comprehensive testing of processStep, we'd need to create proper workflow configs
	// For now, we test that the function handles missing workflow configurations gracefully

	t.Run("missing workflow config", func(t *testing.T) {
		svc, _, _, store := setupService(t)

		// Store run in genie store
		runID := "test-run-id"
		store.Set(runID, &Run{workflowName: "non-existent-workflow"})

		// Create context with timeout to prevent test hanging
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Run processStep - should return quickly due to missing config
		svc.wg.Add(1)
		go svc.processStep(ctx, 0, runID, "non-existent-workflow")

		// Wait for the goroutine to finish
		done := make(chan bool)
		go func() {
			svc.wg.Wait()
			done <- true
		}()

		select {
		case <-done:
			// Test completed successfully - function returned early due to missing config
		case <-time.After(100 * time.Millisecond):
			// Timeout - cancel context and wait
			cancel()
			svc.wg.Wait()
			t.Fatal("processStep should have returned quickly for missing workflow config")
		}

		// The run should still exist in store since processStep doesn't remove it
		_, exists := store.Get(runID)
		require.True(t, exists, "Run should still exist in store after processStep with missing config")
	})
}

func TestMarkRunAsFailed(t *testing.T) {
	fixedTime := time.Date(2023, 1, 1, 16, 0, 0, 0, time.UTC)

	t.Run("successful mark as failed", func(t *testing.T) {
		svc, _, timeProvider, store := setupService(t)

		// Setup a run in the store
		runID := "test-run-id"
		run := &Run{
			workflowName: "test-workflow",
			failed:       false,
		}
		store.Set(runID, run)

		timeProvider.On("Now").Return(fixedTime)

		// Mark run as failed
		svc.markRunAsFailed(runID)

		// Verify the run was marked as failed and end time was set
		runValue, exists := store.Get(runID)
		require.True(t, exists)
		updatedRun := runValue.(*Run)
		require.True(t, updatedRun.failed)
		require.NotNil(t, updatedRun.end)
		require.Equal(t, fixedTime, *updatedRun.end)

		timeProvider.AssertExpectations(t)
	})
}
