package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"os"

	"golang.org/x/oauth2"
)

type OAuthGoogleConfig struct {
	GoogleLoginConfig oauth2.Config
}

func GoogleOAuthConfig() *oauth2.Config {
	var AppConfig = OAuthGoogleConfig{
		GoogleLoginConfig: oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     oauth2.Endpoint{AuthURL: "https://accounts.google.com/o/oauth2/auth", TokenURL: "https://oauth2.googleapis.com/token"},
		},
	}

	return &AppConfig.GoogleLoginConfig
}

func GenerateRandomState() string {
	b := make([]byte, 32)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	return state
}
