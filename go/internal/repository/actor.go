// Package repositoryはリポジトリ層を提供する
// Query結果をModelに変換し、データアクセスを抽象化する
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ActorRepositoryはアクターのリポジトリ
type ActorRepository struct {
	q *query.Queries
}

// NewActorRepositoryはActorRepositoryを生成する
func NewActorRepository(q *query.Queries) *ActorRepository {
	return &ActorRepository{q: q}
}

// WithTxはトランザクションを設定したActorRepositoryを返す
func (r *ActorRepository) WithTx(tx *sql.Tx) *ActorRepository {
	return &ActorRepository{q: r.q.WithTx(tx)}
}

// FindByIDはIDでアクターを取得する
func (r *ActorRepository) FindByID(ctx context.Context, id model.ActorID) (*model.Actor, error) {
	row, err := r.q.GetActorByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toActorModel(row), nil
}

// FindByUserIDはユーザーIDでアクターを取得する
func (r *ActorRepository) FindByUserID(ctx context.Context, userID model.UserID) (*model.Actor, error) {
	row, err := r.q.GetActorByUserID(ctx, uuid.UUID(userID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toActorModel(row), nil
}

// CreateActorInputはアクター作成の入力パラメータ
type CreateActorInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
}

// Createはアクターを作成する
func (r *ActorRepository) Create(ctx context.Context, input CreateActorInput) (*model.Actor, error) {
	row, err := r.q.CreateActor(ctx, query.CreateActorParams{
		UserID:    uuid.UUID(input.UserID),
		ProfileID: uuid.UUID(input.ProfileID),
	})
	if err != nil {
		return nil, err
	}
	return toActorModel(row), nil
}

// toActorModelはquery.Actorをmodel.Actorに変換するパッケージ非公開の
// 自由関数。SessionRepositoryがJOINで取得したactor行をActorRepository
// なしで変換できるように、メソッドではなく自由関数にしている。
func toActorModel(row query.Actor) *model.Actor {
	return &model.Actor{
		ID:        model.ActorID(row.ID),
		UserID:    model.UserID(row.UserID),
		ProfileID: model.ProfileID(row.ProfileID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
