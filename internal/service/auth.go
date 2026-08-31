package service

import (
	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/internal/repository"
	"EquiliLearn/pkg/bcrypt"
	"EquiliLearn/pkg/email"
	"EquiliLearn/pkg/jwt"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type IAuthUsecase interface {
	Register(ctx context.Context, param model.UserRegister) error
	Login(ctx context.Context, param model.UserLogin) (string, error)
	GoogleLogin(state string) string
	GoogleCallback(ctx context.Context, code string) (string, error)
	RequestResetPassword(ctx context.Context, userEmail string) error
	ResetPassword(ctx context.Context, token string, newPassword string) error
}

type AuthUsecase struct {
	Jwt            *jwt.JWT
	Bcrypt         bcrypt.IBcrypt
	Config         *oauth2.Config
	UserRepository repository.IUserRepository
}

func NewAuthUsecase(jwt *jwt.JWT, bcrypt bcrypt.IBcrypt, oAuth2 *oauth2.Config, userRepository repository.IUserRepository) *AuthUsecase {
	return &AuthUsecase{
		Jwt:            jwt,
		Bcrypt:         bcrypt,
		Config:         oAuth2,
		UserRepository: userRepository,
	}
}

func (u *AuthUsecase) Register(ctx context.Context, a model.UserRegister) error {

	a.Email = strings.ToLower(strings.TrimSpace(a.Email))

	if !strings.HasSuffix(a.Email, "@gmail.com") {
		return errors.New("Please use a valid @gmail.com email address.")
	}
	existingUser, _ := u.UserRepository.GetUserByEmail(ctx, a.Email)
	if existingUser != nil {
		return errors.New("Email is already in use")
	}

	a.Email = strings.ToLower(a.Email)

	if a.Password != a.ConfirmPassword {
		return errors.New("Invalid Password")
	}

	hashedPassword, err := u.Bcrypt.GenerateHash(a.Password)
	if err != nil {
		return err
	}

	user := entity.User{
		Id:       uuid.New(),
		Nama:     a.Nama,
		UserName: a.UserName,
		Email:    a.Email,
		Password: hashedPassword,
		Role:     model.UserRoleUser,
	}

	err = u.UserRepository.CreateUser(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (u *AuthUsecase) Login(ctx context.Context, a model.UserLogin) (string, error) {
	user, err := u.UserRepository.GetUserByEmail(ctx, a.Email)
	if err != nil {
		return "", err
	}

	if user == nil {
		return "", errors.New("Email or password is wrong")
	}

	err = u.Bcrypt.ValidatePassword(user.Password, a.Password)
	if err != nil {
		return "", err
	}

	token, err := u.Jwt.GenerateJWT(user.Id.String(), user.Role)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (u *AuthUsecase) GoogleLogin(state string) string {
	return u.Config.AuthCodeURL(state)
}

func (u *AuthUsecase) GoogleCallback(ctx context.Context, code string) (string, error) {
	token, err := u.Config.Exchange(ctx, code)
	if err != nil {
		return "", errors.New("Error" + err.Error())
	}

	client := u.Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", errors.New("Error" + err.Error())
	}

	defer resp.Body.Close()

	var googleUser model.GoogleUser
	err = json.NewDecoder(resp.Body).Decode(&googleUser)
	if err != nil {
		return "", errors.New("Error" + err.Error())
	}

	user, err := u.UserRepository.GetUserByEmail(ctx, googleUser.Email)
	if err != nil {
		return "", nil
	}

	if err == nil {
		user = &entity.User{
			Id:       uuid.New(),
			Email:    googleUser.Email,
			Password: "",
			Role:     model.UserRoleUser,
		}

		err = u.UserRepository.CreateUser(ctx, *user)
		if err != nil {
			return "", err
		}
	}

	jwtToken, err := u.Jwt.GenerateJWT(user.Id.String(), user.Role)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

func (u *AuthUsecase) RequestResetPassword(ctx context.Context, userEmail string) error {
	userEmail = strings.ToLower(userEmail)

	user, err := u.UserRepository.GetUserByEmail(ctx, userEmail)
	if err != nil {
		return err
	}

	if user == nil {
		return nil
	}

	resetToken := uuid.New().String()
	expired := time.Now().Add(15 * time.Minute)

	err = u.UserRepository.SaveResetToken(ctx, user.Id, resetToken, expired)
	if err != nil {
		return err
	}

	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/api/v1"
	}
	resetLink := baseURL + "/reset-password?token=" + resetToken

	err = email.SendEmail(
		user.Email,
		"Reset Password",
		"Link to reset your password :\n"+resetLink,
	)

	if err != nil {
		return err
	}

	return nil
}

func (u *AuthUsecase) ResetPassword(ctx context.Context, token string, newPassword string) error {
	if newPassword == "" {
		return errors.New("Password cannot be empty")
	}

	if len(newPassword) < 8 {
		return errors.New("Password must be at least 8 characters long")
	}

	user, err := u.UserRepository.GetUserByResetToken(ctx, token)
	if err != nil {
		return err
	}

	hashedPassword, err := u.Bcrypt.GenerateHash(newPassword)
	if err != nil {
		return err
	}

	err = u.UserRepository.UpdatePassword(ctx, user.Id, hashedPassword)
	if err != nil {
		return err
	}

	err = u.UserRepository.ClearResetToken(ctx, user.Id)
	if err != nil {
		return err
	}

	return nil
}
