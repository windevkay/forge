package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/windevkay/forge/flho/internal/workflows"
)

type config struct {
	port           int
	workflowConfig string
}

type application struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	config     config
	logger     *slog.Logger
	workflows  *workflows.ConfigStore
	wg         sync.WaitGroup
}

func (a *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", a.healthcheck)

	return mux
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "PORT", 4000, "HTTP server port")
	flag.StringVar(&cfg.workflowConfig, "WORKFLOWS", "", "Path to workflow config YAML")
	flag.Parse()

	workflowConfigStore, err := workflows.NewConfigStoreFromFile(cfg.workflowConfig)
	if err != nil {
		log.Fatal("error loading workflow configurations", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := application{
		ctx:        ctx,
		cancelFunc: cancel,
		config:     cfg,
		logger:     slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		workflows:  workflowConfigStore,
	}

	err = app.serve()
	if err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}
}
