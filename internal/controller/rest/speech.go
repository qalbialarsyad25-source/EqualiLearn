package rest

import (
	"net/http"
	"strconv"

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
