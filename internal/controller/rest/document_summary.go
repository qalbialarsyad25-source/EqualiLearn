package rest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"EquiliLearn/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SummarizeDocument handles multipart file upload (PDF, PPT/PPTX, TXT) and generates an AI summary with Gemini
func (r *V1) SummarizeDocument(c *gin.Context) {
	var req model.SummarizeDocumentRequest
	if err := c.ShouldBind(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "File is required (multipart field 'file')")
		return
	}

	// Max 30MB file size limit
	if fileHeader.Size > 30*1024*1024 {
		RespondError(c, http.StatusBadRequest, "File size exceeds the maximum limit of 30MB")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to process file contents")
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Extract optional authenticated user
	userID := r.getOptionalUserID(c)

	ctx := c.Request.Context()
	result, err := r.service.DocumentSummaryService.SummarizeDocument(ctx, req, fileBytes, fileHeader.Filename, contentType, userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Summarization failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Document summarized successfully",
		"data":    result,
	})
}

// SummarizeText handles direct text/lecture notes summarization
func (r *V1) SummarizeText(c *gin.Context) {
	var req model.SummarizeTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		RespondError(c, http.StatusBadRequest, "Text content is required and cannot be empty")
		return
	}

	userID := r.getOptionalUserID(c)

	ctx := c.Request.Context()
	result, err := r.service.DocumentSummaryService.SummarizeText(ctx, req, userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Summarization failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Text summarized successfully",
		"data":    result,
	})
}

// GetDocumentSummaries returns paginated summary history for the authenticated or guest user
func (r *V1) GetDocumentSummaries(c *gin.Context) {
	var userId uuid.UUID
	userIdVal, exists := c.Get("userId")
	if exists {
		if uid, ok := userIdVal.(uuid.UUID); ok {
			userId = uid
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	pagination := model.Pagination{
		Page:  page,
		Limit: limit,
	}

	ctx := c.Request.Context()
	items, total, err := r.service.DocumentSummaryService.GetSummariesByUserID(ctx, userId, pagination)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to retrieve summary history")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  items,
		"total": total,
		"pagination": gin.H{
			"page":  pagination.Page,
			"limit": pagination.Limit,
		},
	})
}

// GetDocumentSummaryByID returns a single detailed document summary by its ID
func (r *V1) GetDocumentSummaryByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid document summary ID format")
		return
	}

	ctx := c.Request.Context()
	summary, err := r.service.DocumentSummaryService.GetSummaryByID(ctx, id)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "Failed to retrieve document summary")
		return
	}
	if summary == nil {
		RespondError(c, http.StatusNotFound, "Document summary not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": summary,
	})
}

// DeleteDocumentSummary deletes a summary owned by the authenticated user
func (r *V1) DeleteDocumentSummary(c *gin.Context) {
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
		RespondError(c, http.StatusBadRequest, "Invalid document summary ID format")
		return
	}

	ctx := c.Request.Context()
	err = r.service.DocumentSummaryService.DeleteSummary(ctx, id, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			RespondError(c, http.StatusNotFound, "Document summary not found or unauthorized")
			return
		}
		RespondError(c, http.StatusInternalServerError, "Failed to delete document summary")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document summary deleted successfully"})
}

// UpdateDocumentSummary allows updating an existing summary (title, summary, key_points, explanation)
func (r *V1) UpdateDocumentSummary(c *gin.Context) {
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
		RespondError(c, http.StatusBadRequest, "Invalid document summary ID format")
		return
	}

	var req model.UpdateDocumentSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	ctx := c.Request.Context()
	updated, err := r.service.DocumentSummaryService.UpdateSummary(ctx, id, userId, req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || strings.Contains(err.Error(), "not found") {
			RespondError(c, http.StatusNotFound, "Document summary not found")
			return
		}
		if strings.Contains(err.Error(), "unauthorized") {
			RespondError(c, http.StatusForbidden, "Unauthorized to update this document summary")
			return
		}
		RespondError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to update document summary: %v", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Document summary updated successfully",
		"data":    updated,
	})
}

// Helper to extract optional authenticated user ID from context or header
func (r *V1) getOptionalUserID(c *gin.Context) *uuid.UUID {
	userIdVal, exists := c.Get("userId")
	if exists {
		if uid, ok := userIdVal.(uuid.UUID); ok {
			return &uid
		}
	}
	return nil
}
