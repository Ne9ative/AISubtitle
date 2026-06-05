package main

import (
	"context"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// AppInfo renvoie le nom de l'app (binding témoin pour valider le pont Go↔UI).
func (a *App) AppInfo() string {
	return "AI Subtitle Pro"
}
