package rest

import (
	"EquiliLearn/internal/controller/delivery"

	"github.com/gin-gonic/gin"
)

func NewRouter(app *gin.Engine, v1 *V1, speechWSHandler *delivery.SpeechWSHandler, chatWSHandler *delivery.ChatWSHandler) {
	api := app.Group("/api/v1")
	{
		// Authentication endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/register", v1.Register)
			auth.POST("/login", v1.Login)
			auth.POST("/logout", v1.Authentication, v1.Logout)
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
			speech.GET("/history", v1.OptionalAuthentication, v1.GetTranscriptionHistory)
			speech.DELETE("/history/:id", v1.Authentication, v1.DeleteTranscription)

			// Text-to-Speech (TTS)
			speech.POST("/synthesize", v1.OptionalAuthentication, v1.SynthesizeSpeech)
			speech.POST("/text-to-speech", v1.OptionalAuthentication, v1.SynthesizeSpeech)
			speech.GET("/voices", v1.GetTTSVoices)
			speech.GET("/tts/history", v1.OptionalAuthentication, v1.GetTTSHistory)
			speech.DELETE("/tts/history/:id", v1.Authentication, v1.DeleteTTSHistory)
		}

		// AI Document & Presentation Summarization REST endpoints (Gemini)
		documents := api.Group("/documents")
		{
			documents.POST("/summarize", v1.OptionalAuthentication, v1.SummarizeDocument)
			documents.POST("/summarize-text", v1.OptionalAuthentication, v1.SummarizeText)
			documents.GET("/history", v1.OptionalAuthentication, v1.GetDocumentSummaries)
			documents.GET("/:id", v1.GetDocumentSummaryByID)
			documents.PUT("/:id", v1.Authentication, v1.UpdateDocumentSummary)
			documents.PATCH("/:id", v1.Authentication, v1.UpdateDocumentSummary)
			documents.DELETE("/:id", v1.Authentication, v1.DeleteDocumentSummary)
		}

		// Unified Activity History REST endpoints (Document Summaries, STT, TTS)
		history := api.Group("/history")
		{
			history.GET("", v1.OptionalAuthentication, v1.GetAllHistory)
			history.GET("/all", v1.OptionalAuthentication, v1.GetAllHistory)
			history.GET("/stats", v1.OptionalAuthentication, v1.GetHistoryStats)
			history.DELETE("/clear", v1.Authentication, v1.ClearAllHistory)
			history.DELETE("/:type/:id", v1.Authentication, v1.DeleteHistoryItem)
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
