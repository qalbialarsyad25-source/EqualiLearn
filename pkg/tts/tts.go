package tts

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// TTSRequest defines parameters for a text-to-speech synthesis request.
type TTSRequest struct {
	Text       string `json:"text"`
	Model      string `json:"model"`       // Voice model, e.g. "aura-asteria-en", "aura-luna-en"
	Encoding   string `json:"encoding"`    // "mp3", "linear16", "opus", "aac", "flac"
	Container  string `json:"container"`   // "none", "wav"
	SampleRate int    `json:"sample_rate"` // e.g. 24000, 48000, 16000
}

// TTSAudioResult contains the synthesized audio payload and metadata.
type TTSAudioResult struct {
	AudioBytes  []byte
	ContentType string
	Model       string
	Format      string
}

// VoiceInfo describes an available voice model from the provider.
type VoiceInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Gender      string `json:"gender"`
	Language    string `json:"language"`
	Description string `json:"description"`
	SampleRate  int    `json:"sample_rate"`
}

// ITTSClient is the contract interface for text-to-speech providers.
type ITTSClient interface {
	Synthesize(ctx context.Context, req TTSRequest) (*TTSAudioResult, error)
	GetVoices() []VoiceInfo
}

// ==========================================
// Deepgram Speak API Implementation
// ==========================================

// Deepgram Aura Voice Catalog
var defaultDeepgramVoices = []VoiceInfo{
	{
		ID:          "aura-asteria-en",
		Name:        "Asteria (English - US)",
		Gender:      "Female",
		Language:    "en-US",
		Description: "Warm, confident, conversational American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-luna-en",
		Name:        "Luna (English - US)",
		Gender:      "Female",
		Language:    "en-US",
		Description: "Friendly, casual, upbeat American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-stella-en",
		Name:        "Stella (English - US)",
		Gender:      "Female",
		Language:    "en-US",
		Description: "Professional, articulate, polished American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-athena-en",
		Name:        "Athena (English - UK)",
		Gender:      "Female",
		Language:    "en-GB",
		Description: "Sophisticated, warm British English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-hera-en",
		Name:        "Hera (English - US)",
		Gender:      "Female",
		Language:    "en-US",
		Description: "Mature, authoritative, clear American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-orion-en",
		Name:        "Orion (English - US)",
		Gender:      "Male",
		Language:    "en-US",
		Description: "Approachable, storytelling, natural American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-arcas-en",
		Name:        "Arcas (English - US)",
		Gender:      "Male",
		Language:    "en-US",
		Description: "Polished, calm, instructional American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-perseus-en",
		Name:        "Perseus (English - US)",
		Gender:      "Male",
		Language:    "en-US",
		Description: "Casual, youthful, conversational American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-angus-en",
		Name:        "Angus (English - Ireland)",
		Gender:      "Male",
		Language:    "en-IE",
		Description: "Rich, rhythmic Irish English accent",
		SampleRate:  24000,
	},
	{
		ID:          "aura-orpheus-en",
		Name:        "Orpheus (English - US)",
		Gender:      "Male",
		Language:    "en-US",
		Description: "Deep, resonant, confident American English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-helios-en",
		Name:        "Helios (English - UK)",
		Gender:      "Male",
		Language:    "en-GB",
		Description: "Cultured, crisp British English voice",
		SampleRate:  24000,
	},
	{
		ID:          "aura-zeus-en",
		Name:        "Zeus (English - US)",
		Gender:      "Male",
		Language:    "en-US",
		Description: "Commanding, deep, powerful American English voice",
		SampleRate:  24000,
	},
}

type DeepgramTTSClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewDeepgramTTSClient(apiKey string) *DeepgramTTSClient {
	return &DeepgramTTSClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *DeepgramTTSClient) Synthesize(ctx context.Context, req TTSRequest) (*TTSAudioResult, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	model := req.Model
	if model == "" {
		model = "aura-asteria-en"
	}

	encoding := strings.ToLower(req.Encoding)
	if encoding == "" {
		encoding = "mp3"
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.deepgram.com",
		Path:   "/v1/speak",
	}

	q := u.Query()
	q.Set("model", model)
	if encoding != "" {
		q.Set("encoding", encoding)
	}
	if req.Container != "" {
		q.Set("container", req.Container)
	}
	if req.SampleRate > 0 {
		q.Set("sample_rate", fmt.Sprintf("%d", req.SampleRate))
	}
	u.RawQuery = q.Encode()

	requestBody, err := json.Marshal(map[string]string{
		"text": text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode request payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Token "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute deepgram tts request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepgram tts returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio response body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		if encoding == "wav" || req.Container == "wav" {
			contentType = "audio/wav"
		} else if encoding == "opus" {
			contentType = "audio/ogg"
		} else if encoding == "aac" {
			contentType = "audio/aac"
		} else {
			contentType = "audio/mpeg"
		}
	}

	return &TTSAudioResult{
		AudioBytes:  audioData,
		ContentType: contentType,
		Model:       model,
		Format:      encoding,
	}, nil
}

func (c *DeepgramTTSClient) GetVoices() []VoiceInfo {
	return defaultDeepgramVoices
}

// ==========================================
// Mock / Simulation TTS Client (Offline/Dev)
// ==========================================

type MockTTSClient struct{}

func NewMockTTSClient() *MockTTSClient {
	return &MockTTSClient{}
}

// Synthesize in Mock generates a short valid PCM WAV sine wave tone so that clients/browsers receive real playable audio.
func (m *MockTTSClient) Synthesize(ctx context.Context, req TTSRequest) (*TTSAudioResult, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	model := req.Model
	if model == "" {
		model = "aura-asteria-en"
	}

	// Generate a 1.5-second pleasant beep/tone in standard PCM WAV format (16000Hz, 16-bit mono)
	sampleRate := 16000
	durationSeconds := 1.5
	numSamples := int(float64(sampleRate) * durationSeconds)

	pcmData := make([]byte, numSamples*2)
	frequency := 440.0 // A4 note

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		// Slight frequency modulation to feel like speech cadence
		currentFreq := frequency + 50.0*math.Sin(2.0*math.Pi*3.0*t)
		sample := math.Sin(2.0*math.Pi*currentFreq*t) * 0.4
		// Apply fade in and fade out envelope to avoid click sounds
		envelope := 1.0
		if t < 0.1 {
			envelope = t / 0.1
		} else if t > durationSeconds-0.1 {
			envelope = (durationSeconds - t) / 0.1
		}
		sampleVal := int16(sample * envelope * 32767.0)

		binary.LittleEndian.PutUint16(pcmData[i*2:i*2+2], uint16(sampleVal))
	}

	wavBytes := createWAVHeader(sampleRate, 1, 16, len(pcmData))
	wavBytes = append(wavBytes, pcmData...)

	return &TTSAudioResult{
		AudioBytes:  wavBytes,
		ContentType: "audio/wav",
		Model:       model,
		Format:      "wav",
	}, nil
}

func (m *MockTTSClient) GetVoices() []VoiceInfo {
	return defaultDeepgramVoices
}

func createWAVHeader(sampleRate, numChannels, bitsPerSample, dataSize int) []byte {
	buf := new(bytes.Buffer)

	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	chunkSize := 36 + dataSize

	// RIFF chunk
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(chunkSize))
	buf.WriteString("WAVE")

	// fmt sub-chunk
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))             // Subchunk1Size for PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))              // AudioFormat: PCM = 1
	_ = binary.Write(buf, binary.LittleEndian, uint16(numChannels))    // NumChannels
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))     // SampleRate
	_ = binary.Write(buf, binary.LittleEndian, uint32(byteRate))       // ByteRate
	_ = binary.Write(buf, binary.LittleEndian, uint16(blockAlign))     // BlockAlign
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))  // BitsPerSample

	// data sub-chunk
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))

	return buf.Bytes()
}

// ==========================================
// Factory Provider Initializer
// ==========================================

func NewTTSClient() ITTSClient {
	provider := strings.ToLower(os.Getenv("TTS_PROVIDER"))
	deepgramKey := os.Getenv("DEEPGRAM_API_KEY")

	if (provider == "deepgram" || provider == "") && deepgramKey != "" {
		return NewDeepgramTTSClient(deepgramKey)
	}

	return NewMockTTSClient()
}
