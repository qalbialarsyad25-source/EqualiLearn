package rest

import (
	"net/http"
	"os"

	"EquiliLearn/internal/model"
	oauth "EquiliLearn/pkg/Oauth"

	"github.com/gin-gonic/gin"
)

func (r *V1) Register(c *gin.Context) {
	var registerRequest model.UserRegister

	err := c.ShouldBindBodyWithJSON(&registerRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = r.validator.Struct(registerRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	err = r.usecase.AuthUsecase.Register(ctx, registerRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user registered successfully"})
}

func (r *V1) ForgotPassword(c *gin.Context) {
	var req model.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request invalid",
		})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := r.usecase.AuthUsecase.RequestResetPassword(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Jika email terdaftar, link reset akan dikirimkan",
	})
}

func (r *V1) ResetPassword(c *gin.Context) {
	var req model.ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request invalid",
		})
		return
	}

	if err := r.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := r.usecase.AuthUsecase.ResetPassword(c.Request.Context(), req.Token, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Gagal reset password: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "password berhasil diubah",
	})
}

func (r *V1) Login(c *gin.Context) {
	var loginRequest model.UserLogin

	err := c.ShouldBindBodyWithJSON(&loginRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = r.validator.Struct(loginRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	token, err := r.usecase.AuthUsecase.Login(ctx, loginRequest)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (r *V1) LoginGoogle(c *gin.Context) {
	state := oauth.GenerateRandomState()

	c.SetCookie(
		"google_state",
		state,
		3600,
		"/",
		"",
		os.Getenv("APP_ENV") != "development",
		true,
	)

	url := r.usecase.AuthUsecase.GoogleLogin(state)
	c.Redirect(http.StatusFound, url)
}

func (r *V1) CallbackGoogle(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")

	oauth2state, err := c.Cookie("google_state")
	if err != nil || state != oauth2state {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid oauth state"})
		return
	}

	ctx := c.Request.Context()
	token, err := r.usecase.AuthUsecase.GoogleCallback(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
