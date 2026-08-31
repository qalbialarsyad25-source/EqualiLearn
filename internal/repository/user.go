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

type IUserRepository interface {
	CreateUser(ctx context.Context, user entity.User) error
	GetUser(ctx context.Context, pagination model.Pagination) ([]entity.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	SaveResetToken(ctx context.Context, id uuid.UUID, token string, expired time.Time) error
	GetUserByResetToken(ctx context.Context, token string) (*entity.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, password string) error
	ClearResetToken(ctx context.Context, id uuid.UUID) error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db}
}

func (r *UserRepository) CreateUser(ctx context.Context, user entity.User) error {
	return r.db.WithContext(ctx).Create(&user).Error
}

func (r *UserRepository) GetUser(ctx context.Context, pagination model.Pagination) ([]entity.User, error) {
	var users []entity.User
	err := r.db.WithContext(ctx).
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Order("created_at DESC").
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&entity.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *UserRepository) SaveResetToken(ctx context.Context, id uuid.UUID, token string, expired time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"reset_token":         token,
		"reset_token_expired": expired,
	}).Error
}

func (r *UserRepository) GetUserByResetToken(ctx context.Context, token string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("reset_token = ? AND reset_token_expired > ?", token, time.Now()).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid or expired reset token")
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, password string) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", id).Update("password", password).Error
}

func (r *UserRepository) ClearResetToken(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entity.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"reset_token":         "",
		"reset_token_expired": nil,
	}).Error
}
