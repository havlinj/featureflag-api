package app

type App struct {
	Server Server
}

func New () *App {
	return &App{}
} 

func (a *App) Run() error {
	panic("unimplemented")
}