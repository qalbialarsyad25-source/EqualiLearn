package rest

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"EquiliLearn/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetTranscriptionHistory handles retrieving paginated STT history for logged-in user
func (r *V1) GetTranscriptionHistory(c *gin.Context) {
	userIdVal, exists := c.Get("userId")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	pagination := model.Pagination{
		Page:  page,
		Limit: limit,
	}

	ctx := c.Request.Context()
	history, err := r.service.SpeechService.GetTranscriptionHistory(ctx, userId, pagination)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to retrieve transcription history")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": history,
		"pagination": gin.H{
			"page":  pagination.Page,
			"limit": pagination.Limit,
		},
	})
}

// DeleteTranscription handles deleting a specific transcription record
func (r *V1) DeleteTranscription(c *gin.Context) {
	userIdVal, exists := c.Get("userId")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid transcription ID format")
		return
	}

	ctx := c.Request.Context()
	err = r.service.SpeechService.DeleteTranscription(ctx, id, userId)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete transcription")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "transcription deleted successfully"})
}

// SynthesizeSpeech converts input text to speech using Deepgram TTS and returns audio stream or JSON
func (r *V1) SynthesizeSpeech(c *gin.Context) {
	var req model.SynthesizeSpeechRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		// Fallback to query parameters if text is passed in URL query for GET/POST quick calls
		req.Text = c.Query("text")
		req.Voice = c.Query("voice")
		req.Format = c.Query("format")
		if req.Text == "" {
			RespondError(c, http.StatusBadRequest, "Text field is required")
			return
		}
	}

	if strings.TrimSpace(req.Text) == "" {
		RespondError(c, http.StatusBadRequest, "Text cannot be empty")
		return
	}

	// Optional authenticated user
	userID := r.getOptionalUserID(c)

	ctx := c.Request.Context()
	output, err := r.service.SpeechService.SynthesizeSpeech(ctx, req, userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Speech synthesis failed: %v", err))
		return
	}

	// Check if client requested JSON output (e.g. Base64 payload)
	responseType := strings.ToLower(c.Query("response_type"))
	acceptHeader := c.GetHeader("Accept")
	if responseType == "json" || (strings.Contains(acceptHeader, "application/json") && !strings.Contains(acceptHeader, "audio/")) {
		b64Audio := base64.StdEncoding.EncodeToString(output.AudioData)
		c.JSON(http.StatusOK, model.SynthesizeSpeechJsonResponse{
			AudioBase64:        b64Audio,
			ContentType:        output.ContentType,
			Voice:              output.Voice,
			Format:             output.Format,
			DurationEstimateMs: output.DurationEstimateMs,
		})
		return
	}

	// Direct binary audio streaming
	filename := fmt.Sprintf("speech_%s.%s", output.Voice, output.Format)
	c.Header("Content-Type", output.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
	c.Header("Content-Length", strconv.Itoa(len(output.AudioData)))
	c.Header("Accept-Ranges", "bytes")
	c.Header("X-Voice-Model", output.Voice)
	c.Header("X-Estimated-Duration-Ms", strconv.FormatInt(output.DurationEstimateMs, 10))

	c.Data(http.StatusOK, output.ContentType, output.AudioData)
}

// GetTTSVoices returns the catalog of available Deepgram Text-to-Speech voices
func (r *V1) GetTTSVoices(c *gin.Context) {
	ctx := c.Request.Context()
	voices := r.service.SpeechService.GetAvailableVoices(ctx)

	c.JSON(http.StatusOK, gin.H{
		"data":  voices,
		"count": len(voices),
	})
}

// GetTTSHistory handles retrieving paginated text-to-speech generation history for logged-in user
func (r *V1) GetTTSHistory(c *gin.Context) {
	userIdVal, exists := c.Get("userId")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	pagination := model.Pagination{
		Page:  page,
		Limit: limit,
	}

	ctx := c.Request.Context()
	history, total, err := r.service.SpeechService.GetTTSHistory(ctx, userId, pagination)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to retrieve TTS history")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  history,
		"total": total,
		"pagination": gin.H{
			"page":  pagination.Page,
			"limit": pagination.Limit,
		},
	})
}

// DeleteTTSHistory handles deleting a specific TTS record
func (r *V1) DeleteTTSHistory(c *gin.Context) {
	userIdVal, exists := c.Get("userId")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}

	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid TTS history ID format")
		return
	}

	ctx := c.Request.Context()
	err = r.service.SpeechService.DeleteTTSHistory(ctx, id, userId)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to delete TTS history")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "TTS history deleted successfully"})
}

// SummarizeSpeech generates an AI structured summary and key takeaways from a speech transcript or transcription ID
func (r *V1) SummarizeSpeech(c *gin.Context) {
	var req model.SummarizeSpeechRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	if req.TranscriptionID == nil && strings.TrimSpace(req.Text) == "" {
		RespondError(c, http.StatusBadRequest, "Either transcription_id or text is required")
		return
	}

	userID := r.getOptionalUserID(c)
	ctx := c.Request.Context()

	summary, err := r.service.SpeechService.SummarizeTranscription(ctx, req, userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Speech summarization failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Speech summarized successfully",
		"data":    summary,
	})
}

// SummarizeSpeechAudio handles multipart audio file upload (MP3, WAV, M4A, OGG, WebM, FLAC) and generates an AI summary with Gemini
func (r *V1) SummarizeSpeechAudio(c *gin.Context) {
	var req model.SummarizeSpeechAudioRequest
	if err := c.ShouldBind(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Audio file is required (multipart field 'file')")
		return
	}

	// Max 30MB file size limit
	if fileHeader.Size > 30*1024*1024 {
		RespondError(c, http.StatusBadRequest, "Audio file size exceeds the maximum limit of 30MB")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to read uploaded audio file")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to process audio file contents")
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		ext := strings.ToLower(fileHeader.Filename)
		switch {
		case strings.HasSuffix(ext, ".mp3"):
			contentType = "audio/mp3"
		case strings.HasSuffix(ext, ".wav"):
			contentType = "audio/wav"
		case strings.HasSuffix(ext, ".ogg"):
			contentType = "audio/ogg"
		case strings.HasSuffix(ext, ".m4a"):
			contentType = "audio/m4a"
		case strings.HasSuffix(ext, ".flac"):
			contentType = "audio/flac"
		case strings.HasSuffix(ext, ".webm"):
			contentType = "audio/webm"
		default:
			contentType = "audio/mp3"
		}
	}

	userID := r.getOptionalUserID(c)
	ctx := c.Request.Context()

	summary, err := r.service.SpeechService.SummarizeSpeechAudio(ctx, req, fileBytes, fileHeader.Filename, contentType, userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Audio summarization failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Audio speech summarized successfully",
		"data":    summary,
	})
}

// ExportSpeechSummaryByID exports a stored speech summary to PDF, TXT, or MD
func (r *V1) ExportSpeechSummaryByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid speech summary ID format")
		return
	}

	format := c.DefaultQuery("format", "pdf")
	ctx := c.Request.Context()

	result, err := r.service.SpeechService.ExportSpeechSummary(ctx, id, format)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to export speech summary: %v", err))
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Length", strconv.Itoa(len(result.Data)))
	c.Data(http.StatusOK, result.ContentType, result.Data)
}

// ExportSpeechSummaryDirect generates a speech summary on-the-fly and directly streams PDF or Text
func (r *V1) ExportSpeechSummaryDirect(c *gin.Context) {
	var req model.SummarizeSpeechRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	if req.TranscriptionID == nil && strings.TrimSpace(req.Text) == "" {
		RespondError(c, http.StatusBadRequest, "Either transcription_id or text is required")
		return
	}

	format := c.DefaultQuery("format", "pdf")
	userID := r.getOptionalUserID(c)
	ctx := c.Request.Context()

	result, err := r.service.SpeechService.ExportSpeechSummaryDirect(ctx, req, format, userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to export speech summary: %v", err))
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", result.Filename))
	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Length", strconv.Itoa(len(result.Data)))
	c.Data(http.StatusOK, result.ContentType, result.Data)
}
