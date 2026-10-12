package http

import "smart-fridge-backend/internal/transport/http/controllers"

type Handler struct {
	*controllers.AuthController
}

func NewHandler(auth *controllers.AuthController) *Handler {
	return &Handler{
		AuthController: auth,
	}
}
