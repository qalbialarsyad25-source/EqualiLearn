package jwt

import (
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type IJWT interface {
	GenerateJWT(UserId string, role string) (string, error)
	ValidateToken(tokenString string) (string, string, error)
}

type JWT struct {
	SecretKey   string
	ExpiredTime time.Time
}

func NewJWT() *JWT {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	expStr := os.Getenv("JWT_EXPIRED_TIME")
	exp, err := strconv.Atoi(expStr)
	if err != nil {
		exp = 24
	}
	expiredTime := time.Now().Add(time.Duration(exp) * time.Hour)

	return &JWT{
		SecretKey:   secretKey,
		ExpiredTime: expiredTime,
	}
}

type Claims struct {
	UserId string
	Role   string
	jwt.RegisteredClaims
}

func (j *JWT) GenerateJWT(userId string, role string) (string, error) {
	claims := Claims{
		UserId: userId,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(j.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (j *JWT) ValidateToken(tokenString string) (string, string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.SecretKey), nil
	})

	if err != nil {
		return "", "", err
	}

	if !token.Valid {
		return "", "", err
	}

	return claims.UserId, claims.Role, nil
}
