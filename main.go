package main

import (
	"context"
	"embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"goFarmacia/api"
	"goFarmacia/backend"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	db := backend.GetDbInstance()
	gmailSvc := backend.NewGmailService(db)
	bancolombiasSvc := backend.NewBancolombiaService(db)
	driveSvc := backend.NewDriveBackupService(db)
	outlookSvc := backend.NewOutlookService(db)

	ctx := context.Background()
	db.Startup(ctx)

	// startServices is guarded by Once so it is safe to call from both
	// the main goroutine (connected at startup) and the reconnect watcher
	// (connected later after a race or a runtime disconnection).
	var startOnce sync.Once
	startServices := func() {
		startOnce.Do(func() {
			gmailSvc.Startup(ctx)
			bancolombiasSvc.Startup(ctx)
			driveSvc.Startup(ctx)
			outlookSvc.Startup(ctx)
		})
	}

	// Start the background reconnect watcher before launching the HTTP server.
	// It handles two cases:
	//   1. Startup race: PostgreSQL still initialising → quick retries every 5 s
	//      for 60 s, then steady 30 s checks.
	//   2. Runtime disconnection: detects a lost ping and reconnects automatically.
	db.StartReconnectWatcher(ctx, startServices)

	if !db.IsSetupMode() {
		startServices()
	}

	e := api.NewRouter(db, gmailSvc, bancolombiasSvc, driveSvc, outlookSvc, assets)

	port := os.Getenv("PORT")
	if port == "" {
		port = "6969"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: e,
	}

	go func() {
		log.Printf("goFarmacia HTTP server en http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Apagando servidor...")
	bancolombiasSvc.Shutdown()
	driveSvc.Shutdown()
	db.Close()

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("error al apagar: %v", err)
	}
	log.Println("Servidor apagado correctamente.")
}
