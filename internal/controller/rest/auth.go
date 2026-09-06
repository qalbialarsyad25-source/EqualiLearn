package rest

import (
	"EquiliLearn/internal/model"
	"EquiliLearn/pkg/Oauth"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func (r *V1) Register(c *gin.Context) {
	var registerRequest model.UserRegister

	if err := c.ShouldBindBodyWithJSON(&registerRequest); err != nil {
		RespondValidationError(c, err)
		return
	}

	// If confirm_password is not sent in simpler payloads, default it to password
	if registerRequest.ConfirmPassword == "" {
		registerRequest.ConfirmPassword = registerRequest.Password
	}

	if err := r.validator.Struct(registerRequest); err != nil {
		RespondValidationError(c, err)
		return
	}

	ctx := c.Request.Context()
	err := r.service.AuthService.Register(ctx, registerRequest)
	if err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user registered successfully"})
}

func (r *V1) ForgotPassword(c *gin.Context) {
	var req model.ForgotPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	if err := r.validator.Struct(req); err != nil {
		RespondValidationError(c, err)
		return
	}

	err := r.service.AuthService.RequestResetPassword(c.Request.Context(), req.Email)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If email is registered, a reset link will be sent",
	})
}

func (r *V1) ResetPassword(c *gin.Context) {
	var req model.ResetPasswordRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	if err := r.validator.Struct(req); err != nil {
		RespondValidationError(c, err)
		return
	}

	err := r.service.AuthService.ResetPassword(c.Request.Context(), req.Token, req.Password)
	if err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "password changed successfully",
	})
}

func (r *V1) Login(c *gin.Context) {
	var loginRequest model.UserLogin

	if err := c.ShouldBindBodyWithJSON(&loginRequest); err != nil {
		RespondValidationError(c, err)
		return
	}

	if err := r.validator.Struct(loginRequest); err != nil {
		RespondValidationError(c, err)
		return
	}

	ctx := c.Request.Context()
	token, err := r.service.AuthService.Login(ctx, loginRequest)
	if err != nil {
		RespondError(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (p *V1) LoginGoogle(c *gin.Context) {
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

	url := p.service.AuthService.GoogleLogin(state)
	c.Redirect(http.StatusFound, url)
}

func (p *V1) CallbackGoogle(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")

	oauth2state, err := c.Cookie("google_state")
	if err != nil || state != oauth2state {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid oauth state"})
		return
	}

	ctx := c.Request.Context()
	token, err := p.service.AuthService.GoogleCallback(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
