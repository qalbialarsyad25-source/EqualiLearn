package server

import (
	"github.com/gin-gonic/gin"
)

func Start() *gin.Engine {
	app := gin.Default()

	return app
}
