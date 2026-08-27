package repository

import (
	"EquiliLearn/internal/entity"
	"EquiliLearn/internal/model"
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IUserRepository interface {
	CreateUser(ctx context.Context, User entity.User) error
	GetUser(ctx context.Context, pagination model.Pagination) ([]entity.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db}
}

func (p *UserRepository) CreateUser(ctx context.Context, User entity.User) error {
	err := gorm.G[entity.User](p.db).Create(ctx, &User)
	if err != nil {
		return err
	}

	return nil
}

func (p *UserRepository) GetUser(ctx context.Context, pagination model.Pagination) ([]entity.User, error) {
	User, err := gorm.G[entity.User](p.db).
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Order("Created_At DESC").
		Find(ctx)
	if err != nil {
		return nil, err
	}

	return User, nil
}

func (p *UserRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	rows, err := gorm.G[entity.User](p.db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}

	if rows == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
