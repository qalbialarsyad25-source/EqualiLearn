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
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid form parameters: %v", err)})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required (form field 'file')"})
		return
	}

	// Max 30MB file size limit
	if fileHeader.Size > 30*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds maximum limit of 30MB"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file content"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("summarization failed: %v", err)})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request payload: %v", err)})
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text field is required and cannot be empty"})
		return
	}

	userID := r.getOptionalUserID(c)

	ctx := c.Request.Context()
	result, err := r.service.DocumentSummaryService.SummarizeText(ctx, req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("summarization failed: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Text summarized successfully",
		"data":    result,
	})
}

// GetDocumentSummaries returns paginated summary history for the authenticated user
func (r *V1) GetDocumentSummaries(c *gin.Context) {
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
	items, total, err := r.service.DocumentSummaryService.GetSummariesByUserID(ctx, userId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve summary history"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid summary id"})
		return
	}

	ctx := c.Request.Context()
	summary, err := r.service.DocumentSummaryService.GetSummaryByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve document summary"})
		return
	}
	if summary == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document summary not found"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid summary id"})
		return
	}

	ctx := c.Request.Context()
	err = r.service.DocumentSummaryService.DeleteSummary(ctx, id, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "document summary not found or unauthorized"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document summary deleted successfully"})
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
