package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// UserRepositoryはユーザーのリポジトリ
type UserRepository struct {
	q *query.Queries
}

// NewUserRepositoryはUserRepositoryを生成する
func NewUserRepository(q *query.Queries) *UserRepository {
	return &UserRepository{q: q}
}

// WithTxはトランザクションを設定したUserRepositoryを返す
func (r *UserRepository) WithTx(tx *sql.Tx) *UserRepository {
	return &UserRepository{q: r.q.WithTx(tx)}
}

// FindByIDはIDでユーザーを取得する
func (r *UserRepository) FindByID(ctx context.Context, id model.UserID) (*model.User, error) {
	row, err := r.q.GetUserByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toUserModel(row), nil
}

// FindByEmailはメールアドレスでユーザーを取得する
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toUserModel(row), nil
}

// UpdatePasswordByEmailはメールアドレスでユーザーのパスワードを更新する
func (r *UserRepository) UpdatePasswordByEmail(ctx context.Context, email string, passwordDigest string) error {
	return r.q.UpdatePasswordByEmail(ctx, query.UpdatePasswordByEmailParams{
		Email:          email,
		PasswordDigest: passwordDigest,
	})
}

// ExistsByEmailはメールアドレスでユーザーの存在を確認する
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.q.ExistsUserByEmail(ctx, email)
}

// CreateUserInputはユーザー作成の入力パラメータ
type CreateUserInput struct {
	Email          string
	PasswordDigest string
	Locale         string
	TimeZone       string
}

// Createはユーザーを作成する
func (r *UserRepository) Create(ctx context.Context, input CreateUserInput) (*model.User, error) {
	row, err := r.q.CreateUser(ctx, query.CreateUserParams{
		Email:          input.Email,
		PasswordDigest: input.PasswordDigest,
		Locale:         input.Locale,
		TimeZone:       input.TimeZone,
	})
	if err != nil {
		return nil, err
	}
	return toUserModel(row), nil
}

// toUserModelはquery.Userをmodel.Userに変換するパッケージ非公開の
// 自由関数。SessionRepositoryがJOINで取得したuser行をUserRepository
// なしで変換できるように、メソッドではなく自由関数にしている。
func toUserModel(row query.User) *model.User {
	return &model.User{
		ID:             model.UserID(row.ID),
		Email:          row.Email,
		PasswordDigest: row.PasswordDigest,
		Locale:         row.Locale,
		TimeZone:       row.TimeZone,
		SignedUpAt:     row.SignedUpAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}
