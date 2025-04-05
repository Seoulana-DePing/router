package app

import (
	"fmt"
	"os"

	"github.com/router/common/log"
	"github.com/router/config"
	"github.com/router/router"
)

type App struct {
	config       *config.Config
	stop chan struct{}
	router *router.Router
	log  log.Logger
}

func NewApp(cfg *config.Config) *App {
	app := &App{
		stop: make(chan struct{}),
		log:  log.New("moudule", "cmd/app"),
	}
	app.router = router.NewRouter(cfg)
	return app
}

func (a *App) Run() {
	a.router.Run()
}

func (a *App) Wait() {
	a.log.Info("Server started ")
	<-a.stop
	os.Exit(1)
}

func (a *App) Stop() {
	fmt.Println()
	a.log.Info("Server stopped")
	a.stop <- struct{}{}
}
