package rest

import (
	"errors"
	"net/http"
	"strconv"

	"EquiliLearn/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetAllHistory retrieves unified chronological history across Document Summaries, STT, and TTS.
func (r *V1) GetAllHistory(c *gin.Context) {
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
	historyType := c.Query("type")
	search := c.Query("search")
	if search == "" {
		search = c.Query("q")
	}

	query := model.UnifiedHistoryQuery{
		Page:   page,
		Limit:  limit,
		Type:   historyType,
		Search: search,
	}

	ctx := c.Request.Context()
	resp, err := r.service.HistoryService.GetAllHistory(ctx, userId, query)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to retrieve unified history")
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetHistoryStats retrieves aggregate statistics of user activities.
func (r *V1) GetHistoryStats(c *gin.Context) {
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

	ctx := c.Request.Context()
	stats, err := r.service.HistoryService.GetHistoryStats(ctx, userId)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to retrieve history statistics")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}

// DeleteHistoryItem deletes an individual activity record by type and ID.
func (r *V1) DeleteHistoryItem(c *gin.Context) {
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

	itemType := c.Param("type")
	idParam := c.Param("id")

	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid record ID format")
		return
	}

	ctx := c.Request.Context()
	err = r.service.HistoryService.DeleteHistoryItem(ctx, userId, itemType, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondError(c, http.StatusNotFound, "History item not found or unauthorized")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Failed to delete history item")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "History item deleted successfully"})
}

// ClearAllHistory removes history records by category or for all categories.
func (r *V1) ClearAllHistory(c *gin.Context) {
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

	historyType := c.DefaultQuery("type", "all")

	ctx := c.Request.Context()
	err := r.service.HistoryService.ClearAllHistory(ctx, userId, historyType)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to clear history")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "History cleared successfully"})
}
