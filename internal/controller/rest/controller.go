package rest

import (
	websocket "EquiliLearn/internal/controller/delivery"
	"EquiliLearn/internal/service"
	"EquiliLearn/pkg/middleware"

	"github.com/go-playground/validator/v10"
)

type V1 struct {
	middleware.IMiddleware
	validator *validator.Validate
	usecase   *service.Usecase
	wsManager *websocket.WSManager
}

func NewV1(middleware middleware.IMiddleware, validator *validator.Validate, usecase *service.Usecase, wsManager *websocket.WSManager) *V1 {
	return &V1{
		IMiddleware: middleware,
		validator:   validator,
		usecase:     usecase,
		wsManager:   wsManager,
	}
}
