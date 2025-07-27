// Package main provides the FLHO (Fast Lightweight HTTP Operations) application.
// This application implements a workflow execution system with HTTP-based steps
// and automatic retry mechanisms.
package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/windevkay/forge/flho/internal/service"
	"github.com/windevkay/forge/flho/internal/workflow"
	"github.com/windevkay/forge/genie/v2"
)

type config struct {
	dataBackupInterval time.Duration // data backup interval for genie (in-memory store)
	port               int           // HTTP Port
	workflowConfig     string        // path to the workflows YAML config
}

type application struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	config     config
	datastore  *genie.Store
	logger     *slog.Logger
	service    *service.WorkflowService
	workflows  *workflow.ConfigStore
	wg         sync.WaitGroup
}
