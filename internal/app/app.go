package app

import (
	"fmt"
	"os"

	websocket "EquiliLearn/internal/controller/delivery"
	"EquiliLearn/internal/controller/rest"
	"EquiliLearn/internal/repository"
	"EquiliLearn/internal/service"
	oauth "EquiliLearn/pkg/Oauth"
	"EquiliLearn/pkg/bcrypt"
	server "EquiliLearn/pkg/gin"
	"EquiliLearn/pkg/jwt"
	"EquiliLearn/pkg/middleware"
	"EquiliLearn/pkg/postgres"

	"github.com/go-playground/validator/v10"
)

func Run() {
	db := postgres.StartPostgres()
	repo := repository.NewRepository(db)

	jwtService := jwt.NewJWT()
	bcryptService := bcrypt.NewBcrypt()
	oauthConfig := oauth.GoogleOAuthConfig()
	wsManager := websocket.NewWSManager()

	usecase := service.NewUsecase(jwtService, bcryptService, oauthConfig, repo, wsManager)

	mid := middleware.NewMiddleware(jwtService)
	val := validator.New()

	v1 := rest.NewV1(mid, val, usecase, wsManager)

	app := server.Start()

	// Register WebSocket endpoint
	wsHandler := websocket.NewWSHandler(wsManager)
	app.GET("/ws", wsHandler.HandleWS)

	// Register REST API endpoints
	rest.NewRouter(app, v1, wsManager)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	if err := app.Run(":" + port); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
