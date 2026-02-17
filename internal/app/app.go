package app

import "github.com/jan-havlin-dev/featureflag-api/internal/app/server"

type App struct {
	Server server.Server
}

func New () *App {
	return &App{}
} 

func (a *App) Run() error {
	panic("unimplemented")
}