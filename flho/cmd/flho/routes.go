package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", app.healthcheck)
	mux.HandleFunc("/initiateWorkflow", app.initiateWorkflow)
	mux.HandleFunc("/updateWorkflowRun", app.updateWorkflow)
	mux.HandleFunc("/completeWorkflowRun", app.completeWorkflow)

	return mux
}
