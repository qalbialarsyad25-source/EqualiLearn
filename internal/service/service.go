package service

import (
	"EquiliLearn/internal/repository"
	"EquiliLearn/pkg/bcrypt"
	"EquiliLearn/pkg/jwt"
	"EquiliLearn/pkg/stt"
	"EquiliLearn/pkg/tts"

	"golang.org/x/oauth2"
)

type Service struct {
	AuthService   IAuthService
	SpeechService ISpeechService
}

func NewService(jwt *jwt.JWT, bcrypt bcrypt.IBcrypt, oauth *oauth2.Config, repository *repository.Repository, sttClient stt.ISTTClient, ttsClient tts.ITTSClient) *Service {
	return &Service{
		AuthService:   NewAuthService(jwt, bcrypt, oauth, repository.UserRepository),
		SpeechService: NewSpeechService(sttClient, ttsClient, repository.TranscriptionRepository),
	}
}
