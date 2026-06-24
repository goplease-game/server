// Package main ...
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goplease-game/server/ws"
)


func main() {
	config := parseArgs()
	log.Printf("server config: host: %s, port: %s, rwTimeout: %s\n\n",
		config.host, config.port, config.rwTimeout.String())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit,
		syscall.SIGTERM,
		syscall.SIGHUP,  // kill -SIGHUP
		syscall.SIGINT,  // kill -SIGINT or Ctrl+c
		syscall.SIGQUIT, // kill -SIGQUIT
	)

	hub := ws.NewHub()
	gs := ws.NewGameServer(hub)

	go hub.Run()
	go gs.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/play/", hub.ServeWS)

	addr := net.JoinHostPort(config.host, config.port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  config.rwTimeout,
		WriteTimeout: config.rwTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("[goplease] server running at %s", addr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		log.Fatalf("[goplease] server error: %v", err)

	case sig := <-quit:
		log.Printf("[goplease] %v signal received...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := server.Shutdown(ctx)
		if err != nil {
			_ = server.Close()
		}

		log.Println("[goplease] bye")
	}
}
