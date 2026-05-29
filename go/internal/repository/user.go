package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// UserRepository はユーザーのリポジトリ
type UserRepository struct {
	q *query.Queries
}

// NewUserRepository はUserRepositoryを生成する
func NewUserRepository(q *query.Queries) *UserRepository {
	return &UserRepository{q: q}
}

// WithTx はトランザクションを設定したUserRepositoryを返す
func (r *UserRepository) WithTx(tx *sql.Tx) *UserRepository {
	return &UserRepository{q: r.q.WithTx(tx)}
}

// FindByID はIDでユーザーを取得する
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

// FindByEmail はメールアドレスでユーザーを取得する
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

// UpdatePasswordByEmail はメールアドレスでユーザーのパスワードを更新する
func (r *UserRepository) UpdatePasswordByEmail(ctx context.Context, email string, passwordDigest string) error {
	return r.q.UpdatePasswordByEmail(ctx, query.UpdatePasswordByEmailParams{
		Email:          email,
		PasswordDigest: passwordDigest,
	})
}

// ExistsByEmail はメールアドレスでユーザーの存在を確認する
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.q.ExistsUserByEmail(ctx, email)
}

// CreateUserInput はユーザー作成の入力パラメータ
type CreateUserInput struct {
	Email          string
	PasswordDigest string
	Locale         string
	TimeZone       string
}

// Create はユーザーを作成する
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

// toUserModel converts a query.User row into a model.User. It is a
// package-private free function so SessionRepository's JOIN-based auth lookup
// can reuse the conversion without instantiating a UserRepository.
//
// [Ja] toUserModel は query.User を model.User に変換するパッケージ非公開の
// 自由関数。SessionRepository が JOIN で取得した user 行を UserRepository
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
