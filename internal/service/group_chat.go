package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"EquiliLearn/internal/repository"

	"github.com/google/uuid"
)

type IGroupChatService interface {
	SearchUsers(ctx context.Context, query string) ([]model.UserSearchResponse, error)
	CreateGroup(ctx context.Context, creatorID uuid.UUID, req model.CreateGroupRequest) (*model.GroupDetailResponse, error)
	GetUserGroups(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.GroupResponse, int64, error)
	GetGroupDetail(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (*model.GroupDetailResponse, error)
	AddMember(ctx context.Context, groupID uuid.UUID, requesterID uuid.UUID, req model.AddGroupMemberRequest) (*model.GroupMemberResponse, error)
	RemoveMember(ctx context.Context, groupID uuid.UUID, requesterID uuid.UUID, targetUserID uuid.UUID) error
	SaveMessage(ctx context.Context, groupID uuid.UUID, senderID uuid.UUID, content string, msgType string) (*model.GroupMessageResponse, error)
	GetGroupMessages(ctx context.Context, groupID uuid.UUID, userID uuid.UUID, pagination model.Pagination) ([]model.GroupMessageResponse, int64, error)
	ValidateGroupMembership(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (bool, string, error)
}

type GroupChatService struct {
	groupRepo repository.IGroupChatRepository
	userRepo  repository.IUserRepository
}

func NewGroupChatService(groupRepo repository.IGroupChatRepository, userRepo repository.IUserRepository) *GroupChatService {
	return &GroupChatService{
		groupRepo: groupRepo,
		userRepo:  userRepo,
	}
}

func (s *GroupChatService) SearchUsers(ctx context.Context, query string) ([]model.UserSearchResponse, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []model.UserSearchResponse{}, nil
	}

	users, err := s.userRepo.SearchUsers(ctx, trimmed, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	res := make([]model.UserSearchResponse, len(users))
	for i, u := range users {
		res[i] = model.UserSearchResponse{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		}
	}
	return res, nil
}

func (s *GroupChatService) CreateGroup(ctx context.Context, creatorID uuid.UUID, req model.CreateGroupRequest) (*model.GroupDetailResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("group name is required")
	}

	// Resolve member IDs from MemberEmails if provided
	resolvedMemberIDs := make([]uuid.UUID, 0, len(req.MemberUserIDs)+len(req.MemberEmails))
	resolvedMemberIDs = append(resolvedMemberIDs, req.MemberUserIDs...)

	for _, email := range req.MemberEmails {
		emailTrimmed := strings.TrimSpace(email)
		if emailTrimmed == "" {
			continue
		}
		u, err := s.userRepo.GetUserByEmail(ctx, emailTrimmed)
		if err == nil && u != nil {
			resolvedMemberIDs = append(resolvedMemberIDs, u.ID)
		}
	}

	groupID := uuid.New()
	groupEntity := &entity.Group{
		ID:          groupID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		CreatedByID: creatorID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.groupRepo.CreateGroup(ctx, groupEntity, resolvedMemberIDs); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return s.GetGroupDetail(ctx, groupID, creatorID)
}

func (s *GroupChatService) GetUserGroups(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]model.GroupResponse, int64, error) {
	pagination.Check()
	groups, total, err := s.groupRepo.GetUserGroups(ctx, userID, pagination)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user groups: %w", err)
	}

	res := make([]model.GroupResponse, len(groups))
	for i, g := range groups {
		var createdByName string
		if g.CreatedBy != nil {
			createdByName = g.CreatedBy.Name
		}

		// Determine user's role in this group
		userRole := "member"
		for _, m := range g.Members {
			if m.UserID == userID {
				userRole = m.Role
				break
			}
		}

		// Fetch last message
		var lastMsgResp *model.GroupMessageResponse
		lastMsg, err := s.groupRepo.GetLastGroupMessage(ctx, g.ID)
		if err == nil && lastMsg != nil {
			senderName := "Unknown"
			senderEmail := ""
			if lastMsg.Sender != nil {
				senderName = lastMsg.Sender.Name
				senderEmail = lastMsg.Sender.Email
			}
			lastMsgResp = &model.GroupMessageResponse{
				ID:          lastMsg.ID,
				GroupID:     lastMsg.GroupID,
				SenderID:    lastMsg.SenderID,
				SenderName:  senderName,
				SenderEmail: senderEmail,
				Content:     lastMsg.Content,
				MessageType: lastMsg.MessageType,
				CreatedAt:   lastMsg.CreatedAt,
			}
		}

		res[i] = model.GroupResponse{
			ID:            g.ID,
			Name:          g.Name,
			Description:   g.Description,
			CreatedByID:   g.CreatedByID,
			CreatedByName: createdByName,
			MemberCount:   len(g.Members),
			UserRole:      userRole,
			LastMessage:   lastMsgResp,
			CreatedAt:     g.CreatedAt,
			UpdatedAt:     g.UpdatedAt,
		}
	}

	return res, total, nil
}

func (s *GroupChatService) GetGroupDetail(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (*model.GroupDetailResponse, error) {
	// Verify user is member of group
	isMember, _, err := s.groupRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied: you are not a member of this group")
	}

	group, err := s.groupRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("group not found")
	}

	var createdByName string
	if group.CreatedBy != nil {
		createdByName = group.CreatedBy.Name
	}

	members := make([]model.GroupMemberResponse, len(group.Members))
	for i, m := range group.Members {
		var name, email string
		if m.User != nil {
			name = m.User.Name
			email = m.User.Email
		}
		members[i] = model.GroupMemberResponse{
			ID:       m.ID,
			GroupID:  m.GroupID,
			UserID:   m.UserID,
			Name:     name,
			Email:    email,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		}
	}

	return &model.GroupDetailResponse{
		ID:            group.ID,
		Name:          group.Name,
		Description:   group.Description,
		CreatedByID:   group.CreatedByID,
		CreatedByName: createdByName,
		Members:       members,
		CreatedAt:     group.CreatedAt,
		UpdatedAt:     group.UpdatedAt,
	}, nil
}

func (s *GroupChatService) AddMember(ctx context.Context, groupID uuid.UUID, requesterID uuid.UUID, req model.AddGroupMemberRequest) (*model.GroupMemberResponse, error) {
	// Verify requester is member (or admin)
	isMember, _, err := s.groupRepo.IsMember(ctx, groupID, requesterID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied: only group members can invite other users")
	}

	// Resolve target user
	var targetUser *entity.User
	if req.UserID != nil {
		targetUser, err = s.userRepo.GetUserByID(ctx, *req.UserID)
	} else if strings.TrimSpace(req.Email) != "" {
		targetUser, err = s.userRepo.GetUserByEmail(ctx, strings.TrimSpace(req.Email))
	} else {
		return nil, errors.New("either user_id or email is required")
	}

	if err != nil || targetUser == nil {
		return nil, errors.New("target user not found")
	}

	// Check if already member
	alreadyMember, _, _ := s.groupRepo.IsMember(ctx, groupID, targetUser.ID)
	if alreadyMember {
		return nil, errors.New("user is already a member of this group")
	}

	role := "member"
	if req.Role == "admin" {
		role = "admin"
	}

	newMember := &entity.GroupMember{
		ID:       uuid.New(),
		GroupID:  groupID,
		UserID:   targetUser.ID,
		Role:     role,
		JoinedAt: time.Now(),
	}

	if err := s.groupRepo.AddMember(ctx, newMember); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	return &model.GroupMemberResponse{
		ID:       newMember.ID,
		GroupID:  newMember.GroupID,
		UserID:   newMember.UserID,
		Name:     targetUser.Name,
		Email:    targetUser.Email,
		Role:     newMember.Role,
		JoinedAt: newMember.JoinedAt,
	}, nil
}

func (s *GroupChatService) RemoveMember(ctx context.Context, groupID uuid.UUID, requesterID uuid.UUID, targetUserID uuid.UUID) error {
	isMember, reqRole, err := s.groupRepo.IsMember(ctx, groupID, requesterID)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("access denied: you are not a member of this group")
	}

	// Self-leaving or Admin kicking someone
	if requesterID != targetUserID && reqRole != "admin" {
		return errors.New("access denied: only group admins can remove other members")
	}

	return s.groupRepo.RemoveMember(ctx, groupID, targetUserID)
}

func (s *GroupChatService) SaveMessage(ctx context.Context, groupID uuid.UUID, senderID uuid.UUID, content string, msgType string) (*model.GroupMessageResponse, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, errors.New("message content cannot be empty")
	}

	// Validate sender is member
	isMember, _, err := s.groupRepo.IsMember(ctx, groupID, senderID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("access denied: cannot send message to a group you do not belong to")
	}

	if msgType == "" {
		msgType = "text"
	}

	msgID := uuid.New()
	msg := &entity.GroupMessage{
		ID:          msgID,
		GroupID:     groupID,
		SenderID:    senderID,
		Content:     trimmed,
		MessageType: msgType,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.groupRepo.SaveMessage(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	sender, _ := s.userRepo.GetUserByID(ctx, senderID)
	senderName := "Unknown"
	senderEmail := ""
	if sender != nil {
		senderName = sender.Name
		senderEmail = sender.Email
	}

	return &model.GroupMessageResponse{
		ID:          msg.ID,
		GroupID:     msg.GroupID,
		SenderID:    msg.SenderID,
		SenderName:  senderName,
		SenderEmail: senderEmail,
		Content:     msg.Content,
		MessageType: msg.MessageType,
		CreatedAt:   msg.CreatedAt,
	}, nil
}

func (s *GroupChatService) GetGroupMessages(ctx context.Context, groupID uuid.UUID, userID uuid.UUID, pagination model.Pagination) ([]model.GroupMessageResponse, int64, error) {
	isMember, _, err := s.groupRepo.IsMember(ctx, groupID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isMember {
		return nil, 0, errors.New("access denied: you are not a member of this group")
	}

	pagination.Check()
	messages, total, err := s.groupRepo.GetGroupMessages(ctx, groupID, pagination)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get group messages: %w", err)
	}

	res := make([]model.GroupMessageResponse, len(messages))
	for i, m := range messages {
		senderName := "Unknown"
		senderEmail := ""
		if m.Sender != nil {
			senderName = m.Sender.Name
			senderEmail = m.Sender.Email
		}
		res[i] = model.GroupMessageResponse{
			ID:          m.ID,
			GroupID:     m.GroupID,
			SenderID:    m.SenderID,
			SenderName:  senderName,
			SenderEmail: senderEmail,
			Content:     m.Content,
			MessageType: m.MessageType,
			CreatedAt:   m.CreatedAt,
		}
	}

	return res, total, nil
}

func (s *GroupChatService) ValidateGroupMembership(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (bool, string, error) {
	return s.groupRepo.IsMember(ctx, groupID, userID)
}
