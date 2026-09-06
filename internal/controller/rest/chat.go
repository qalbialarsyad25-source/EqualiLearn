package rest

import (
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
		RespondError(c, http.StatusInternalServerError, "Failed to search users")
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
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	var req model.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		RespondError(c, http.StatusBadRequest, "Group name is required")
		return
	}

	ctx := c.Request.Context()
	group, err := r.service.GroupChatService.CreateGroup(ctx, userId, req)
	if err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
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
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
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
		RespondError(c, http.StatusInternalServerError, "Failed to retrieve user groups")
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
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid group ID format")
		return
	}

	ctx := c.Request.Context()
	group, err := r.service.GroupChatService.GetGroupDetail(ctx, groupID, userId)
	if err != nil {
		RespondError(c, http.StatusForbidden, err.Error())
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
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid group ID format")
		return
	}

	var req model.AddGroupMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondValidationError(c, err)
		return
	}

	ctx := c.Request.Context()
	member, err := r.service.GroupChatService.AddMember(ctx, groupID, userId, req)
	if err != nil {
		RespondError(c, http.StatusBadRequest, err.Error())
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
		RespondError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	userId, ok := userIdVal.(uuid.UUID)
	if !ok {
		RespondError(c, http.StatusBadRequest, "Invalid user identity")
		return
	}

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid group ID format")
		return
	}

	targetUserIdParam := c.Param("userId")
	targetUserId, err := uuid.Parse(targetUserIdParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid target user ID format")
		return
	}

	ctx := c.Request.Context()
	err = r.service.GroupChatService.RemoveMember(ctx, groupID, userId, targetUserId)
	if err != nil {
		RespondError(c, http.StatusForbidden, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed from group successfully"})
}

// GetGroupMessages returns paginated message history for a group
func (r *V1) GetGroupMessages(c *gin.Context) {
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

	groupIDParam := c.Param("id")
	groupID, err := uuid.Parse(groupIDParam)
	if err != nil {
		RespondError(c, http.StatusBadRequest, "Invalid group ID format")
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
		RespondError(c, http.StatusForbidden, err.Error())
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
