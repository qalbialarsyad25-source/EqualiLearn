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

// GetTranscriptionHistory handles retrieving paginated speech transcription history for logged-in user
func (r *V1) GetTranscriptionHistory(c *gin.Context) {
	userIdVal, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve transcription history"})
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid transcription id"})
		return
	}

	ctx := c.Request.Context()
	err = r.service.SpeechService.DeleteTranscription(ctx, id, userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete transcription"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "text field is required"})
			return
		}
	}

	if strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text cannot be empty"})
		return
	}

	ctx := c.Request.Context()
	output, err := r.service.SpeechService.SynthesizeSpeech(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("speech synthesis failed: %v", err)})
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
