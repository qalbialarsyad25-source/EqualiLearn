package rest

import (
	"encoding/base64"
	"fmt"
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
