package service

import (
	"context"
	"testing"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/pkg/stt"
	"EquiliLearn/pkg/tts"

	"github.com/google/uuid"
)

type mockTranscriptionRepo struct {
	transcriptions []entity.Transcription
}

func (m *mockTranscriptionRepo) CreateTranscription(ctx context.Context, transcription *entity.Transcription) error {
	m.transcriptions = append(m.transcriptions, *transcription)
	return nil
}

func (m *mockTranscriptionRepo) GetTranscriptionsByUserID(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.Transcription, error) {
	var results []entity.Transcription
	for _, t := range m.transcriptions {
		if t.UserID != nil && *t.UserID == userID {
			results = append(results, t)
		}
	}
	return results, nil
}

func (m *mockTranscriptionRepo) GetTranscriptionByID(ctx context.Context, id uuid.UUID) (*entity.Transcription, error) {
	for _, t := range m.transcriptions {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockTranscriptionRepo) DeleteTranscription(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	var filtered []entity.Transcription
	for _, t := range m.transcriptions {
		if !(t.ID == id && t.UserID != nil && *t.UserID == userID) {
			filtered = append(filtered, t)
		}
	}
	m.transcriptions = filtered
	return nil
}

func TestSpeechService_StartSTTSession(t *testing.T) {
	mockSTT := stt.NewMockSTTClient()
	mockTTS := tts.NewMockTTSClient()
	repo := &mockTranscriptionRepo{}
	svc := NewSpeechService(mockSTT, mockTTS, repo)

	ctx := context.Background()
	userID := uuid.New()
	cfg := model.STTStreamConfig{
		LanguageCode:   "id-ID",
		SampleRate:     16000,
		Encoding:       "linear16",
		InterimResults: true,
	}

	session, sessionID, err := svc.StartSTTSession(ctx, &userID, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sessionID == "" {
		t.Fatal("expected non-empty sessionID")
	}
	defer session.Close()

	// Send simulated audio chunk
	err = session.SendAudio([]byte{0x01, 0x02, 0x03, 0x04})
	if err != nil {
		t.Fatalf("expected no error sending audio, got %v", err)
	}
}

func TestSpeechService_SaveAndGetHistory(t *testing.T) {
	mockSTT := stt.NewMockSTTClient()
	mockTTS := tts.NewMockTTSClient()
	repo := &mockTranscriptionRepo{}
	svc := NewSpeechService(mockSTT, mockTTS, repo)

	ctx := context.Background()
	userID := uuid.New()
	sessionID := uuid.New().String()

	saved, err := svc.SaveFinalTranscription(ctx, &userID, sessionID, "id-ID", "Halo dunia", 0.98, 1200)
	if err != nil {
		t.Fatalf("failed to save transcription: %v", err)
	}
	if saved == nil || saved.Text != "Halo dunia" {
		t.Fatalf("unexpected saved text: %v", saved)
	}

	history, err := svc.GetTranscriptionHistory(ctx, userID, model.Pagination{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history))
	}
	if history[0].Text != "Halo dunia" {
		t.Errorf("expected text 'Halo dunia', got '%s'", history[0].Text)
	}

	err = svc.DeleteTranscription(ctx, saved.ID, userID)
	if err != nil {
		t.Fatalf("failed to delete transcription: %v", err)
	}

	historyAfterDelete, _ := svc.GetTranscriptionHistory(ctx, userID, model.Pagination{Page: 1, Limit: 10})
	if len(historyAfterDelete) != 0 {
		t.Fatalf("expected 0 history items after deletion, got %d", len(historyAfterDelete))
	}
}

func TestSpeechService_SynthesizeSpeechAndVoices(t *testing.T) {
	mockSTT := stt.NewMockSTTClient()
	mockTTS := tts.NewMockTTSClient()
	repo := &mockTranscriptionRepo{}
	svc := NewSpeechService(mockSTT, mockTTS, repo)

	ctx := context.Background()

	// 1. Test GetAvailableVoices
	voices := svc.GetAvailableVoices(ctx)
	if len(voices) == 0 {
		t.Fatal("expected at least 1 available voice")
	}

	foundAsteria := false
	for _, v := range voices {
		if v.ID == "aura-asteria-en" {
			foundAsteria = true
			break
		}
	}
	if !foundAsteria {
		t.Errorf("expected 'aura-asteria-en' in voice list")
	}

	// 2. Test SynthesizeSpeech
	req := model.SynthesizeSpeechRequest{
		Text:   "Hello from EquiliLearn text to speech engine.",
		Voice:  "aura-asteria-en",
		Format: "wav",
	}

	output, err := svc.SynthesizeSpeech(ctx, req)
	if err != nil {
		t.Fatalf("expected no error synthesizing speech, got %v", err)
	}

	if output == nil {
		t.Fatal("expected non-nil output")
	}

	if len(output.AudioData) == 0 {
		t.Fatal("expected non-empty audio data")
	}

	if output.ContentType != "audio/wav" {
		t.Errorf("expected ContentType 'audio/wav', got %s", output.ContentType)
	}

	if output.Voice != "aura-asteria-en" {
		t.Errorf("expected Voice 'aura-asteria-en', got %s", output.Voice)
	}

	// 3. Test empty text error
	_, err = svc.SynthesizeSpeech(ctx, model.SynthesizeSpeechRequest{Text: ""})
	if err == nil {
		t.Fatal("expected error when synthesizing empty text")
	}
}
