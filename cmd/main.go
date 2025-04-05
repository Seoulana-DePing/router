package main

import (
	"flag"

	"github.com/router/cmd/app"
	"github.com/router/config"
)
var configFlag = flag.String("config", "./config.toml", "configuration toml file path")

func main() {
	config := config.NewConfig(*configFlag)
	app := app.NewApp(config)
	go app.Wait()
	app.Run()
}
