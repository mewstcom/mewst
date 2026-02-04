// Package repository はリポジトリ層を提供する
// Query 結果を Model に変換し、データアクセスを抽象化する
package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ActorRepository はアクターのリポジトリ
type ActorRepository struct {
	queries *query.Queries
}

// NewActorRepository はActorRepositoryを生成する
func NewActorRepository(db query.DBTX) *ActorRepository {
	return &ActorRepository{
		queries: query.New(db),
	}
}

// WithTx はトランザクションを設定したActorRepositoryを返す
func (r *ActorRepository) WithTx(tx *sql.Tx) *ActorRepository {
	return &ActorRepository{
		queries: r.queries.WithTx(tx),
	}
}

// GetByID はIDでアクターを取得する
func (r *ActorRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Actor, error) {
	row, err := r.queries.GetActorByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.Actor{
		ID:        row.ID,
		UserID:    row.UserID,
		ProfileID: row.ProfileID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}

// GetByUserID はユーザーIDでアクターを取得する
func (r *ActorRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.Actor, error) {
	row, err := r.queries.GetActorByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.Actor{
		ID:        row.ID,
		UserID:    row.UserID,
		ProfileID: row.ProfileID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}
