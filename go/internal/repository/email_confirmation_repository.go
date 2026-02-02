package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/query"
)

// EmailConfirmationRepository はメール確認のリポジトリ
type EmailConfirmationRepository struct {
	queries *query.Queries
}

// NewEmailConfirmationRepository はEmailConfirmationRepositoryを生成する
func NewEmailConfirmationRepository(db query.DBTX) *EmailConfirmationRepository {
	return &EmailConfirmationRepository{
		queries: query.New(db),
	}
}

// WithTx はトランザクションを設定したEmailConfirmationRepositoryを返す
func (r *EmailConfirmationRepository) WithTx(tx *sql.Tx) *EmailConfirmationRepository {
	return &EmailConfirmationRepository{
		queries: r.queries.WithTx(tx),
	}
}

// CreateEmailConfirmationParams はメール確認作成のパラメータ
type CreateEmailConfirmationParams struct {
	Email string
	Event model.EmailConfirmationEvent
	Code  string
}

// Create はメール確認を作成する
func (r *EmailConfirmationRepository) Create(ctx context.Context, params CreateEmailConfirmationParams) (*model.EmailConfirmation, error) {
	row, err := r.queries.CreateEmailConfirmation(ctx, query.CreateEmailConfirmationParams{
		Email: params.Email,
		Event: string(params.Event),
		Code:  params.Code,
	})
	if err != nil {
		return nil, err
	}

	return r.toModel(row), nil
}

// GetByID はIDでメール確認を取得する
func (r *EmailConfirmationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.EmailConfirmation, error) {
	row, err := r.queries.GetEmailConfirmationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return r.toModel(row), nil
}

// GetActiveByID は有効期限内かつ未確認のメール確認をIDで取得する
func (r *EmailConfirmationRepository) GetActiveByID(ctx context.Context, id uuid.UUID) (*model.EmailConfirmation, error) {
	row, err := r.queries.GetActiveEmailConfirmationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return r.toModel(row), nil
}

// GetSucceededByID は確認済みのメール確認をIDで取得する
func (r *EmailConfirmationRepository) GetSucceededByID(ctx context.Context, id uuid.UUID) (*model.EmailConfirmation, error) {
	row, err := r.queries.GetSucceededEmailConfirmationByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return r.toModel(row), nil
}

// MarkAsSucceeded はメール確認を成功済みとしてマークする
func (r *EmailConfirmationRepository) MarkAsSucceeded(ctx context.Context, id uuid.UUID) error {
	return r.queries.MarkEmailConfirmationAsSucceeded(ctx, id)
}

// toModel はsqlcの型をドメインモデルに変換する
func (r *EmailConfirmationRepository) toModel(row query.EmailConfirmation) *model.EmailConfirmation {
	var succeededAt *time.Time
	if row.SucceededAt.Valid {
		succeededAt = &row.SucceededAt.Time
	}

	return &model.EmailConfirmation{
		ID:          row.ID,
		Email:       row.Email,
		Event:       model.EmailConfirmationEvent(row.Event),
		Code:        row.Code,
		SucceededAt: succeededAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
