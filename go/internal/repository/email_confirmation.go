package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// EmailConfirmationRepository はメール確認のリポジトリ
type EmailConfirmationRepository struct {
	q *query.Queries
}

// NewEmailConfirmationRepository はEmailConfirmationRepositoryを生成する
func NewEmailConfirmationRepository(q *query.Queries) *EmailConfirmationRepository {
	return &EmailConfirmationRepository{q: q}
}

// WithTx はトランザクションを設定したEmailConfirmationRepositoryを返す
func (r *EmailConfirmationRepository) WithTx(tx *sql.Tx) *EmailConfirmationRepository {
	return &EmailConfirmationRepository{q: r.q.WithTx(tx)}
}

// CreateEmailConfirmationInput はメール確認作成の入力パラメータ
type CreateEmailConfirmationInput struct {
	Email string
	Event model.EmailConfirmationEvent
	Code  string
}

// Create はメール確認を作成する
func (r *EmailConfirmationRepository) Create(ctx context.Context, input CreateEmailConfirmationInput) (*model.EmailConfirmation, error) {
	row, err := r.q.CreateEmailConfirmation(ctx, query.CreateEmailConfirmationParams{
		Email: input.Email,
		Event: string(input.Event),
		Code:  input.Code,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// FindByID はIDでメール確認を取得する
func (r *EmailConfirmationRepository) FindByID(ctx context.Context, id model.EmailConfirmationID) (*model.EmailConfirmation, error) {
	row, err := r.q.GetEmailConfirmationByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindActiveByID は有効期限内かつ未確認のメール確認をIDで取得する
func (r *EmailConfirmationRepository) FindActiveByID(ctx context.Context, id model.EmailConfirmationID) (*model.EmailConfirmation, error) {
	row, err := r.q.GetActiveEmailConfirmationByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindSucceededByID は確認済みのメール確認をIDで取得する
func (r *EmailConfirmationRepository) FindSucceededByID(ctx context.Context, id model.EmailConfirmationID) (*model.EmailConfirmation, error) {
	row, err := r.q.GetSucceededEmailConfirmationByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// Succeed はメール確認を成功済みとしてマークする
func (r *EmailConfirmationRepository) Succeed(ctx context.Context, id model.EmailConfirmationID) error {
	return r.q.UpdateEmailConfirmationSucceededAt(ctx, uuid.UUID(id))
}

// toModel は query.EmailConfirmation を model.EmailConfirmation に変換する
func (r *EmailConfirmationRepository) toModel(row query.EmailConfirmation) *model.EmailConfirmation {
	var succeededAt *time.Time
	if row.SucceededAt.Valid {
		succeededAt = &row.SucceededAt.Time
	}

	return &model.EmailConfirmation{
		ID:          model.EmailConfirmationID(row.ID),
		Email:       row.Email,
		Event:       model.EmailConfirmationEvent(row.Event),
		Code:        row.Code,
		SucceededAt: succeededAt,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
