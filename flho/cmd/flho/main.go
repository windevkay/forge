package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/windevkay/forge/flho/internal/workflow"
	"github.com/windevkay/forge/genie"
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
	workflows  *workflow.ConfigStore
	wg         sync.WaitGroup
}

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", app.healthcheck)

	return mux
}

func main() {
	var cfg config
	const defaultHTTPPort = 4000
	const defaultDataBackupInterval = 10

	flag.IntVar(&cfg.port, "PORT", defaultHTTPPort, "HTTP server port")
	flag.StringVar(&cfg.workflowConfig, "WORKFLOWS", "", "Path to workflow config YAML")
	flag.DurationVar(&cfg.dataBackupInterval, "DBINTRVL", time.Duration(defaultDataBackupInterval), "Data backup interval")
	flag.Parse()

	workflowConfigStore, err := workflow.NewConfigStoreFromFile(cfg.workflowConfig)
	if err != nil {
		log.Fatal("error loading workflow configurations", err.Error())
	}

	dataStore, err := genie.NewStore()
	if err != nil {
		log.Fatal("error setting up datastore", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	app := application{
		ctx:        ctx,
		cancelFunc: cancel,
		config:     cfg,
		datastore:  dataStore,
		logger:     slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		workflows:  workflowConfigStore,
	}

	app.datastore.StartAutoBackup(app.config.dataBackupInterval * time.Minute)

	// monitor for errors in data backup
	go func() {
		for err := range app.datastore.AutoBackupErrors() {
			app.logger.Warn(fmt.Sprintf("error backing up data: %s", err.Error()))
		}
	}()

	err = app.serve()
	if err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}
}
