package handler

import (
	"dfkgo/api/response"
	"dfkgo/entity"
	"dfkgo/errcode"

	authsvc "dfkgo/service/auth"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *authsvc.AuthService
}

func NewAuthHandler(authService *authsvc.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req entity.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, errcode.ErrInvalidEmail)
		return
	}

	if err := h.authService.Register(req.Email, req.Password); err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OKWithMsg(c, "注册成功", nil)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req entity.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithErr(c, errcode.ErrInvalidEmail)
		return
	}

	token, expiresAt, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		response.FailWithErr(c, err)
		return
	}
	response.OK(c, entity.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	})
}
