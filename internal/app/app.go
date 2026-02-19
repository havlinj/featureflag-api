package app

import "context"


type App struct {
	Server Server
}

func NewApp () *App {
	return &App{}
} 

func (a *App) Run(socket string) error {
	panic("unimplemented")
}

func (a *App) Shutdown (ctx context.Context) error {
	panic("unimplemented")
}