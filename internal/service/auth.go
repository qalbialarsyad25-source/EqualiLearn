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

type IAuthService interface {
	Register(ctx context.Context, param model.UserRegister) error
	Login(ctx context.Context, param model.UserLogin) (string, error)
	GoogleLogin(state string) string
	GoogleCallback(ctx context.Context, code string) (string, error)
	RequestResetPassword(ctx context.Context, userEmail string) error
	ResetPassword(ctx context.Context, token string, newPassword string) error
}

type AuthService struct {
	Jwt            *jwt.JWT
	Bcrypt         bcrypt.IBcrypt
	Config         *oauth2.Config
	UserRepository repository.IUserRepository
}

func NewAuthService(jwt *jwt.JWT, bcrypt bcrypt.IBcrypt, oAuth2 *oauth2.Config, userRepository repository.IUserRepository) *AuthService {
	return &AuthService{
		Jwt:            jwt,
		Bcrypt:         bcrypt,
		Config:         oAuth2,
		UserRepository: userRepository,
	}
}

func (u *AuthService) Register(ctx context.Context, a model.UserRegister) error {

	a.Email = strings.ToLower(strings.TrimSpace(a.Email))

	if !strings.HasSuffix(a.Email, "@gmail.com") {
		return errors.New("email harus menggunakan @gmail.com")
	}
	existingUser, _ := u.UserRepository.GetUserByEmail(ctx, a.Email)
	if existingUser != nil {
		return errors.New("email sudah digunakan")
	}

	a.Email = strings.ToLower(a.Email)

	if a.Password != a.ConfirmPassword {
		return errors.New("Password tidak sama")
	}

	hashedPassword, err := u.Bcrypt.GenerateHash(a.Password)
	if err != nil {
		return err
	}

	user := entity.User{
		ID:       uuid.New(),
		Name:     a.Name,
		Email:    a.Email,
		Password: hashedPassword,
	}

	err = u.UserRepository.CreateUser(ctx, user)
	if err != nil {
		return err
	}
	return nil
}

func (u *AuthService) Login(ctx context.Context, a model.UserLogin) (string, error) {
	user, err := u.UserRepository.GetUserByEmail(ctx, a.Email)
	if err != nil {
		return "", err
	}

	if user == nil {
		return "", errors.New("email atau password salah")
	}

	err = u.Bcrypt.ValidatePassword(user.Password, a.Password)
	if err != nil {
		return "", err
	}

	token, err := u.Jwt.GenerateJWT(user.ID.String(), "user")
	if err != nil {
		return "", err
	}

	return token, nil
}

func (u *AuthService) GoogleLogin(state string) string {
	return u.Config.AuthCodeURL(state)
}

func (u *AuthService) GoogleCallback(ctx context.Context, code string) (string, error) {
	token, err := u.Config.Exchange(ctx, code)
	if err != nil {
		return "", errors.New("gagal" + err.Error())
	}

	client := u.Config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", errors.New("Gagal" + err.Error())
	}

	defer resp.Body.Close()

	var googleUser model.GoogleUser
	err = json.NewDecoder(resp.Body).Decode(&googleUser)
	if err != nil {
		return "", errors.New("Gagal" + err.Error())
	}

	user, err := u.UserRepository.GetUserByEmail(ctx, googleUser.Email)
	if err != nil {
		return "", nil
	}

	if err == nil {
		user = &entity.User{
			ID:       uuid.New(),
			Name:     googleUser.Name,
			Email:    googleUser.Email,
			Password: "",
		}

		err = u.UserRepository.CreateUser(ctx, *user)
		if err != nil {
			return "", err
		}
	}

	jwtToken, err := u.Jwt.GenerateJWT(user.ID.String(), user.Role)
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

func (u *AuthService) RequestResetPassword(ctx context.Context, userEmail string) error {
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

	err = u.UserRepository.SaveResetToken(ctx, user.ID, resetToken, expired)
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
		"Klik link berikut untuk reset password:\n"+resetLink,
	)

	if err != nil {
		return err
	}

	return nil
}

func (u *AuthService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	if newPassword == "" {
		return errors.New("password tidak boleh kosong")
	}

	if len(newPassword) < 8 {
		return errors.New("password minimal 8 karakter")
	}

	user, err := u.UserRepository.GetUserByResetToken(ctx, token)
	if err != nil {
		return err
	}

	hashedPassword, err := u.Bcrypt.GenerateHash(newPassword)
	if err != nil {
		return err
	}

	err = u.UserRepository.UpdatePassword(ctx, user.ID, hashedPassword)
	if err != nil {
		return err
	}

	err = u.UserRepository.ClearResetToken(ctx, user.ID)
	if err != nil {
		return err
	}

	return nil
}
