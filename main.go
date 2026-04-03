package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goFarmacia/api"
	"goFarmacia/backend"
)

func main() {
	db := backend.GetDbInstance()
	gmailSvc := backend.NewGmailService(db)
	bancolombiasSvc := backend.NewBancolombiaService(db)
	driveSvc := backend.NewDriveBackupService(db)

	ctx := context.Background()
	db.Startup(ctx)
	gmailSvc.Startup(ctx)
	bancolombiasSvc.Startup(ctx)
	driveSvc.Startup(ctx)

	e := api.NewRouter(db, gmailSvc, bancolombiasSvc, driveSvc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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
