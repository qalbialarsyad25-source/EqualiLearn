package rest

import (
	"EquiliLearn/internal/controller/delivery"

	"github.com/gin-gonic/gin"
)

func NewRouter(app *gin.Engine, v1 *V1, wsHandler *delivery.SpeechWSHandler) {
	// Serve public assets / test demo page
	app.Static("/public", "./public")
	app.StaticFile("/demo", "./public/speech_test.html")

	api := app.Group("/api/v1")
	{
		// Authentication endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/register", v1.Register)
			auth.POST("/login", v1.Login)
			auth.GET("/google/login", v1.LoginGoogle)
			auth.GET("/google/callback", v1.CallbackGoogle)
			auth.POST("/forgot-password", v1.ForgotPassword)
			auth.POST("/reset-password", v1.ResetPassword)
		}

		// Real-time WebSocket endpoints
		ws := api.Group("/ws")
		{
			ws.GET("/speech-to-text", wsHandler.HandleRealtimeSTT)
		}

		// Speech transcription REST endpoints
		speech := api.Group("/speech")
		{
			speech.GET("/history", v1.Authentication, v1.GetTranscriptionHistory)
			speech.DELETE("/history/:id", v1.Authentication, v1.DeleteTranscription)
		}
	}
}