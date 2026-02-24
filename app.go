package main

import (
	"context"
	"fmt"
	"goFarmacia/backend"
)

// App struct
type App struct {
	ctx context.Context
	db  *backend.Db
}

func NewApp(db *backend.Db) *App {
	return &App{
		db: db,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.db.Startup(ctx)
}

// shutdown is called when the app terminates.
func (a *App) shutdown(ctx context.Context) {
	a.db.Log.Info("Cerrando la aplicación y la base de datos...")
	a.db.Close()
	a.db.Log.Info("Base de datos cerrada correctamente.")
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
