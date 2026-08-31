package rest

import (
	"EquiliLearn/internal/service"
	"EquiliLearn/pkg/middleware"

	"github.com/go-playground/validator/v10"
)

type V1 struct {
	middleware.IMiddleware
	validator *validator.Validate
	service   *service.Service
}

func NewV1(middleware middleware.IMiddleware, validator *validator.Validate, service *service.Service) *V1 {
	return &V1{
		IMiddleware: middleware,
		validator:   validator,
		service:     service,
	}
}
