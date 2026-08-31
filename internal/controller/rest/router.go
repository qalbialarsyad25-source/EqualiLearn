package rest

import (
	websocket "EquiliLearn/internal/controller/delivery"

	"github.com/gin-gonic/gin"
)

func NewRouter(app *gin.Engine, v1 *V1, wsManager *websocket.WSManager) {
	api := app.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", v1.Register)
			auth.POST("/login", v1.Login)
			auth.GET("/google/login", v1.LoginGoogle)
			auth.GET("/google/callback", v1.CallbackGoogle)
			auth.POST("/forgot-password", v1.ForgotPassword)
			auth.POST("/reset-password", v1.ResetPassword)
		}
	}
}