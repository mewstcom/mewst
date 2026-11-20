package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// UserProfileRepositoryはユーザープロフィール関連付けのリポジトリ
type UserProfileRepository struct {
	q *query.Queries
}

// NewUserProfileRepositoryはUserProfileRepositoryを生成する
func NewUserProfileRepository(q *query.Queries) *UserProfileRepository {
	return &UserProfileRepository{q: q}
}

// WithTxはトランザクションを設定したUserProfileRepositoryを返す
func (r *UserProfileRepository) WithTx(tx *sql.Tx) *UserProfileRepository {
	return &UserProfileRepository{q: r.q.WithTx(tx)}
}

// CreateUserProfileInputはユーザープロフィール関連付け作成の入力パラメータ
type CreateUserProfileInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
}

// FindByUserIDはユーザーIDでユーザープロフィール関連付けを取得する
func (r *UserProfileRepository) FindByUserID(ctx context.Context, userID model.UserID) (*model.UserProfile, error) {
	row, err := r.q.GetUserProfileByUserID(ctx, uuid.UUID(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByProfileIDはプロフィールIDでユーザープロフィール関連付けを取得する
func (r *UserProfileRepository) FindByProfileID(ctx context.Context, profileID model.ProfileID) (*model.UserProfile, error) {
	row, err := r.q.GetUserProfileByProfileID(ctx, uuid.UUID(profileID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// Createはユーザープロフィール関連付けを作成する
func (r *UserProfileRepository) Create(ctx context.Context, input CreateUserProfileInput) (*model.UserProfile, error) {
	row, err := r.q.CreateUserProfile(ctx, query.CreateUserProfileParams{
		UserID:    uuid.UUID(input.UserID),
		ProfileID: uuid.UUID(input.ProfileID),
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// toModelはquery.UserProfileをmodel.UserProfileに変換する
func (r *UserProfileRepository) toModel(row query.UserProfile) *model.UserProfile {
	return &model.UserProfile{
		ID:        model.UserProfileID(row.ID),
		UserID:    model.UserID(row.UserID),
		ProfileID: model.ProfileID(row.ProfileID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
