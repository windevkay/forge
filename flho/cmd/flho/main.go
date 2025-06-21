package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

type config struct {
	port int
}

type application struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	config     config
	logger     *slog.Logger
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
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	app := application{
		ctx:        ctx,
		cancelFunc: cancel,
		config:     cfg,
		logger:     slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}

	err := app.serve()
	if err != nil {
		app.logger.Error(err.Error())
		os.Exit(1)
	}
}
