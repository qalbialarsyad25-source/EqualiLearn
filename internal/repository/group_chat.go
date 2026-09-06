package repository

import (
	"context"
	"errors"
	"time"

	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IGroupChatRepository interface {
	CreateGroup(ctx context.Context, group *entity.Group, memberUserIDs []uuid.UUID) error
	GetGroupByID(ctx context.Context, groupID uuid.UUID) (*entity.Group, error)
	GetUserGroups(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.Group, int64, error)
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]entity.GroupMember, error)
	AddMember(ctx context.Context, member *entity.GroupMember) error
	RemoveMember(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) error
	IsMember(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (bool, string, error)
	SaveMessage(ctx context.Context, msg *entity.GroupMessage) error
	GetGroupMessages(ctx context.Context, groupID uuid.UUID, pagination model.Pagination) ([]entity.GroupMessage, int64, error)
	GetLastGroupMessage(ctx context.Context, groupID uuid.UUID) (*entity.GroupMessage, error)
}

type GroupChatRepository struct {
	db *gorm.DB
}

func NewGroupChatRepository(db *gorm.DB) *GroupChatRepository {
	return &GroupChatRepository{db: db}
}

// CreateGroup creates the group and adds creator as admin, plus any initial members.
func (r *GroupChatRepository) CreateGroup(ctx context.Context, group *entity.Group, memberUserIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}

		// Add creator as admin
		creatorMember := entity.GroupMember{
			ID:       uuid.New(),
			GroupID:  group.ID,
			UserID:   group.CreatedByID,
			Role:     "admin",
			JoinedAt: time.Now(),
		}
		if err := tx.Create(&creatorMember).Error; err != nil {
			return err
		}

		// Add other initial members
		seen := map[uuid.UUID]bool{group.CreatedByID: true}
		for _, memberID := range memberUserIDs {
			if seen[memberID] {
				continue
			}
			seen[memberID] = true
			m := entity.GroupMember{
				ID:       uuid.New(),
				GroupID:  group.ID,
				UserID:   memberID,
				Role:     "member",
				JoinedAt: time.Now(),
			}
			if err := tx.Create(&m).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *GroupChatRepository) GetGroupByID(ctx context.Context, groupID uuid.UUID) (*entity.Group, error) {
	var group entity.Group
	err := r.db.WithContext(ctx).
		Preload("CreatedBy").
		Preload("Members.User").
		Where("id = ?", groupID).
		First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

func (r *GroupChatRepository) GetUserGroups(ctx context.Context, userID uuid.UUID, pagination model.Pagination) ([]entity.Group, int64, error) {
	var groups []entity.Group
	var total int64

	// Find groups where user is a member
	subQuery := r.db.WithContext(ctx).Model(&entity.GroupMember{}).Select("group_id").Where("user_id = ?", userID)

	countQuery := r.db.WithContext(ctx).Model(&entity.Group{}).Where("id IN (?)", subQuery)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Preload("CreatedBy").
		Preload("Members.User").
		Where("id IN (?)", subQuery).
		Order("updated_at DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&groups).Error
	if err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

func (r *GroupChatRepository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]entity.GroupMember, error) {
	var members []entity.GroupMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("group_id = ?", groupID).
		Order("joined_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, err
	}
	return members, nil
}

func (r *GroupChatRepository) AddMember(ctx context.Context, member *entity.GroupMember) error {
	return r.db.WithContext(ctx).Create(member).Error
}

func (r *GroupChatRepository) RemoveMember(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Delete(&entity.GroupMember{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *GroupChatRepository) IsMember(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (bool, string, error) {
	var member entity.GroupMember
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, member.Role, nil
}

func (r *GroupChatRepository) SaveMessage(ctx context.Context, msg *entity.GroupMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		// Update group's updated_at timestamp
		return tx.Model(&entity.Group{}).Where("id = ?", msg.GroupID).Update("updated_at", msg.CreatedAt).Error
	})
}

func (r *GroupChatRepository) GetGroupMessages(ctx context.Context, groupID uuid.UUID, pagination model.Pagination) ([]entity.GroupMessage, int64, error) {
	var messages []entity.GroupMessage
	var total int64

	query := r.db.WithContext(ctx).Model(&entity.GroupMessage{}).Where("group_id = ?", groupID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Preload("Sender").
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&messages).Error
	if err != nil {
		return nil, 0, err
	}

	// Reverse messages slice so caller receives them in chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, total, nil
}

func (r *GroupChatRepository) GetLastGroupMessage(ctx context.Context, groupID uuid.UUID) (*entity.GroupMessage, error) {
	var msg entity.GroupMessage
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		First(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}
