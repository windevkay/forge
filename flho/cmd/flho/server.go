package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	const readTimeout, writeTimeout = 5, 10

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  readTimeout * time.Second,
		WriteTimeout: writeTimeout * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGQUIT)
		s := <-quit
		app.logger.Info("intercepted signal", "signal", s.String())

		defer app.cancelFunc()

		err := srv.Shutdown(app.ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.Info("...finishing background tasks", "addr", srv.Addr)
		app.cancelFunc()
		app.datastore.StopAutoBackup()
		app.wg.Wait()

		shutdownError <- nil
	}()

	app.logger.Info("starting server", "addr", srv.Addr)

	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.Info("server stopped", "addr", srv.Addr)

	return nil
}
