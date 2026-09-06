package rest

import (
	"EquiliLearn/internal/controller/delivery"

	"github.com/gin-gonic/gin"
)

func NewRouter(app *gin.Engine, v1 *V1, speechWSHandler *delivery.SpeechWSHandler, chatWSHandler *delivery.ChatWSHandler) {
	// Serve public assets / test demo pages
	app.Static("/public", "./public")
	app.StaticFile("/demo", "./public/speech_test.html")
	app.StaticFile("/summary-demo", "./public/document_summary_demo.html")
	app.StaticFile("/chat-demo", "./public/chat_demo.html")

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

		// User discovery endpoints
		users := api.Group("/users")
		{
			users.GET("/search", v1.Authentication, v1.SearchUsers)
		}

		// Real-time WebSocket endpoints
		ws := api.Group("/ws")
		{
			ws.GET("/speech-to-text", speechWSHandler.HandleRealtimeSTT)
			ws.GET("/chat", chatWSHandler.HandleGroupChat)
		}

		// Speech transcription and Text-to-Speech REST endpoints
		speech := api.Group("/speech")
		{
			// Speech-to-Text history
			speech.GET("/history", v1.Authentication, v1.GetTranscriptionHistory)
			speech.DELETE("/history/:id", v1.Authentication, v1.DeleteTranscription)

			// Text-to-Speech (TTS)
			speech.POST("/synthesize", v1.SynthesizeSpeech)
			speech.POST("/text-to-speech", v1.SynthesizeSpeech)
			speech.GET("/voices", v1.GetTTSVoices)
		}

		// AI Document & Presentation Summarization REST endpoints (Gemini)
		documents := api.Group("/documents")
		{
			documents.POST("/summarize", v1.SummarizeDocument)
			documents.POST("/summarize-text", v1.SummarizeText)
			documents.GET("/history", v1.Authentication, v1.GetDocumentSummaries)
			documents.GET("/:id", v1.GetDocumentSummaryByID)
			documents.DELETE("/:id", v1.Authentication, v1.DeleteDocumentSummary)
		}

		// Collaborative Group Chat REST endpoints
		groups := api.Group("/groups")
		{
			groups.POST("", v1.Authentication, v1.CreateGroup)
			groups.GET("", v1.Authentication, v1.GetUserGroups)
			groups.GET("/:id", v1.Authentication, v1.GetGroupDetail)
			groups.POST("/:id/members", v1.Authentication, v1.AddGroupMember)
			groups.DELETE("/:id/members/:userId", v1.Authentication, v1.RemoveGroupMember)
			groups.GET("/:id/messages", v1.Authentication, v1.GetGroupMessages)
		}
	}
}