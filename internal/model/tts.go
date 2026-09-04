package model

// SynthesizeSpeechRequest represents the client request to convert text to speech.
type SynthesizeSpeechRequest struct {
	Text       string `json:"text" binding:"required"`
	Voice      string `json:"voice"`       // Voice model ID (e.g. "aura-asteria-en", "aura-luna-en")
	Format     string `json:"format"`      // "mp3", "wav", "opus", "aac"
	SampleRate int    `json:"sample_rate"` // e.g. 24000, 48000, 16000
}

// TTSAudioOutput represents the synthesized audio data ready to return to callers.
type TTSAudioOutput struct {
	AudioData          []byte `json:"-"`
	ContentType        string `json:"content_type"`
	Voice              string `json:"voice"`
	Format             string `json:"format"`
	DurationEstimateMs int64  `json:"duration_estimate_ms,omitempty"`
}

// TTSVoiceResponse represents metadata of an available TTS voice.
type TTSVoiceResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	Language    string `json:"language"`
	Description string `json:"description"`
	SampleRate  int    `json:"sample_rate"`
}

// SynthesizeSpeechJsonResponse is used when a client explicitly requests JSON encoded output instead of direct binary audio streaming.
type SynthesizeSpeechJsonResponse struct {
	AudioBase64        string `json:"audio_base64"`
	ContentType        string `json:"content_type"`
	Voice              string `json:"voice"`
	Format             string `json:"format"`
	DurationEstimateMs int64  `json:"duration_estimate_ms,omitempty"`
}
