package app

import (
	"fmt"
	"os"

	"EquiliLearn/internal/controller/delivery"
	"EquiliLearn/internal/controller/rest"
	"EquiliLearn/internal/repository"
	"EquiliLearn/internal/service"
	oauth "EquiliLearn/pkg/Oauth"
	"EquiliLearn/pkg/bcrypt"
	server "EquiliLearn/pkg/gin"
	"EquiliLearn/pkg/jwt"
	"EquiliLearn/pkg/middleware"
	"EquiliLearn/pkg/postgres"
	"EquiliLearn/pkg/stt"
	"EquiliLearn/pkg/tts"

	"github.com/go-playground/validator/v10"
)

func Run() {
	db := postgres.StartPostgres()
	repo := repository.NewRepository(db)

	jwtService := jwt.NewJWT()
	bcryptService := bcrypt.NewBcrypt()
	oauthConfig := oauth.GoogleOAuthConfig()
	sttClient := stt.NewSTTClient()
	ttsClient := tts.NewTTSClient()

	svc := service.NewService(jwtService, bcryptService, oauthConfig, repo, sttClient, ttsClient)

	mid := middleware.NewMiddleware(jwtService)
	val := validator.New()

	v1 := rest.NewV1(mid, val, svc)

	wsManager := delivery.NewWSManager()
	speechWSHandler := delivery.NewSpeechWSHandler(wsManager, svc.SpeechService, jwtService)

	app := server.Start()

	// Register REST and WebSocket API endpoints
	rest.NewRouter(app, v1, speechWSHandler)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := app.Run(":" + port); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
