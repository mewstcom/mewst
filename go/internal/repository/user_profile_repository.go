package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// UserProfileRepository はユーザープロフィール関連付けのリポジトリ
type UserProfileRepository struct {
	queries *query.Queries
}

// NewUserProfileRepository はUserProfileRepositoryを生成する
func NewUserProfileRepository(db query.DBTX) *UserProfileRepository {
	return &UserProfileRepository{
		queries: query.New(db),
	}
}

// WithTx はトランザクションを設定したUserProfileRepositoryを返す
func (r *UserProfileRepository) WithTx(tx *sql.Tx) *UserProfileRepository {
	return &UserProfileRepository{
		queries: r.queries.WithTx(tx),
	}
}

// CreateUserProfileParams はユーザープロフィール関連付け作成のパラメータ
type CreateUserProfileParams struct {
	UserID    uuid.UUID
	ProfileID uuid.UUID
}

// GetByUserID はユーザーIDでユーザープロフィール関連付けを取得する
func (r *UserProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error) {
	row, err := r.queries.GetUserProfileByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return r.toModel(row), nil
}

// GetByProfileID はプロフィールIDでユーザープロフィール関連付けを取得する
func (r *UserProfileRepository) GetByProfileID(ctx context.Context, profileID uuid.UUID) (*model.UserProfile, error) {
	row, err := r.queries.GetUserProfileByProfileID(ctx, profileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return r.toModel(row), nil
}

// Create はユーザープロフィール関連付けを作成する
func (r *UserProfileRepository) Create(ctx context.Context, params CreateUserProfileParams) (*model.UserProfile, error) {
	row, err := r.queries.CreateUserProfile(ctx, query.CreateUserProfileParams{
		UserID:    params.UserID,
		ProfileID: params.ProfileID,
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(row), nil
}

// toModel はsqlcの行データをモデルに変換する
func (r *UserProfileRepository) toModel(row query.UserProfile) *model.UserProfile {
	return &model.UserProfile{
		ID:        row.ID,
		UserID:    row.UserID,
		ProfileID: row.ProfileID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
