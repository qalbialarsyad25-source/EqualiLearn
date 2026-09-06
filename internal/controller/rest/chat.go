package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"EquiliLearn/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SearchUsers allows authenticated users to search for peers by email or name or UUID to add them to group chats
func (r *V1) SearchUsers(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusOK, gin.H{
			"data":  []model.UserSearchResponse{},
			"count": 0,
		})
		return
	}

	ctx := c.Request.Context()
	users, err := r.service.GroupChatService.SearchUsers(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to search users: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  users,
		"count": len(users),
	})
}

// CreateGroup creates a new collaborative group chat
func (r *V1) CreateGroup(c *gin.Context) {
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

	var req model.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group name is required"})
		return
	}

	ctx := c.Request.Context()
	group, err := r.service.GroupChatService.CreateGroup(ctx, userId, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create group: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Group created successfully",
		"data":    group,
	})
}

// GetUserGroups returns all groups the authenticated user is a member of
func (r *V1) GetUserGroups(c *gin.Context) {
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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	pagination := model.Pagination{
		Page:  page,
		Limit: limit,
	}

	ctx := c.Request.Context()
	groups, total, err := r.service.GroupChatService.GetUserGroups(ctx, userId, pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get groups: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  groups,
		"total": total,
		"pagination": gin.H{
			"page":  pagination.Page,
			"limit": pagination.Limit,
		},
	})
}

// GetGroupDetail returns detailed info and member roster for a group
func (r *V1) GetGroupDetail(c *gin.Context) {
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

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	ctx := c.Request.Context()
	group, err := r.service.GroupChatService.GetGroupDetail(ctx, groupID, userId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": group,
	})
}

// AddGroupMember adds another user to an existing group by user ID or Email
func (r *V1) AddGroupMember(c *gin.Context) {
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

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	var req model.AddGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request payload: %v", err)})
		return
	}

	ctx := c.Request.Context()
	member, err := r.service.GroupChatService.AddMember(ctx, groupID, userId, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Member added successfully",
		"data":    member,
	})
}

// RemoveGroupMember removes a member from a group (or allows a user to leave a group)
func (r *V1) RemoveGroupMember(c *gin.Context) {
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

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	targetUserIdParam := c.Param("userId")
	targetUserId, err := uuid.Parse(targetUserIdParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target user id"})
		return
	}

	ctx := c.Request.Context()
	err = r.service.GroupChatService.RemoveMember(ctx, groupID, userId, targetUserId)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed from group successfully"})
}

// GetGroupMessages returns paginated message history for a group
func (r *V1) GetGroupMessages(c *gin.Context) {
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

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	pagination := model.Pagination{
		Page:  page,
		Limit: limit,
	}

	ctx := c.Request.Context()
	messages, total, err := r.service.GroupChatService.GetGroupMessages(ctx, groupID, userId, pagination)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  messages,
		"total": total,
		"pagination": gin.H{
			"page":  pagination.Page,
			"limit": pagination.Limit,
		},
	})
}
