package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/ticketing-system/backend/internal/model"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, name, created_at)
		VALUES (:id, :email, :password_hash, :name, :created_at)
	`, user)
	return err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE email = $1", email)
	return &user, err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
	return &user, err
}
