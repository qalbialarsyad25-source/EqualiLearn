package main

import (
	"EquiliLearn/config"
	"EquiliLearn/internal/app"
)

func main() {
	config.NewConfig()
	app.Run()

}
