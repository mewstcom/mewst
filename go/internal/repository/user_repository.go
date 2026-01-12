package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/query"
)

// UserRepository はユーザーのリポジトリ
type UserRepository struct {
	queries *query.Queries
}

// NewUserRepository はUserRepositoryを生成する
func NewUserRepository(db query.DBTX) *UserRepository {
	return &UserRepository{
		queries: query.New(db),
	}
}

// WithTx はトランザクションを設定したUserRepositoryを返す
func (r *UserRepository) WithTx(tx *sql.Tx) *UserRepository {
	return &UserRepository{
		queries: r.queries.WithTx(tx),
	}
}

// GetByID はIDでユーザーを取得する
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row, err := r.queries.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.User{
		ID:             row.ID,
		Email:          row.Email,
		PasswordDigest: row.PasswordDigest,
		Locale:         row.Locale,
		TimeZone:       row.TimeZone,
		SignedUpAt:     row.SignedUpAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// GetByEmail はメールアドレスでユーザーを取得する
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.User{
		ID:             row.ID,
		Email:          row.Email,
		PasswordDigest: row.PasswordDigest,
		Locale:         row.Locale,
		TimeZone:       row.TimeZone,
		SignedUpAt:     row.SignedUpAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}, nil
}

// GetByEmailForSignIn はログイン用にメールアドレスでユーザーを取得する
// パスワード検証に必要な最小限のフィールドのみ取得する
func (r *UserRepository) GetByEmailForSignIn(ctx context.Context, email string) (*model.User, error) {
	row, err := r.queries.GetUserByEmailForSignIn(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.User{
		ID:             row.ID,
		Email:          row.Email,
		PasswordDigest: row.PasswordDigest,
	}, nil
}

// UpdatePasswordByEmail はメールアドレスでユーザーのパスワードを更新する
func (r *UserRepository) UpdatePasswordByEmail(ctx context.Context, email string, passwordDigest string) error {
	return r.queries.UpdatePasswordByEmail(ctx, query.UpdatePasswordByEmailParams{
		Email:          email,
		PasswordDigest: passwordDigest,
	})
}
