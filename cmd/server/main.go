// Package main is the entry point of the server application.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ognev-dev/goplease/app"
	"github.com/ognev-dev/goplease/app/service"
	"github.com/ognev-dev/goplease/server"
	"github.com/ognev-dev/goplease/tracing"
	"github.com/ognev-dev/goplease/worker"
)

func main() {
	conf := app.Config()
	ctx, cancelCtx := context.WithCancel(context.Background())
	tracer, err := tracing.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = worker.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGTERM,
		syscall.SIGHUP,  // kill -SIGHUP
		syscall.SIGINT,  // kill -SIGINT or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT
	)

	db, err := app.NewDB(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = app.MigrateDB(ctx, db)
	if err != nil {
		db.Close()
		log.Fatal(err)
	}

	defer db.Close() // log.Fatal will exit, and `defer db.Close()` will not run (gocritic)

	services := service.New(db, tracer)
	srv := server.New(services, tracer)

	go func() {
		<-quit
		cancelCtx()
		err := srv.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Println(conf.App.Name + " (" + conf.App.Version + ") serving at " + conf.Server.Host + ":" + conf.Server.Port)

	if conf.Server.AutocertHosts != "" {
		err = srv.ListenAndServeTLS("", "")
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Println(err.Error())
		return
	}

	log.Println(conf.App.Name + " (" + conf.App.Version + ") server closed")
}
