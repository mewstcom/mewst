package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/internal/model"
	"github.com/mewstcom/mewst/internal/query"
)

// SessionRepository はセッションのリポジトリ
type SessionRepository struct {
	queries *query.Queries
}

// NewSessionRepository はSessionRepositoryを生成する
func NewSessionRepository(db query.DBTX) *SessionRepository {
	return &SessionRepository{
		queries: query.New(db),
	}
}

// WithTx はトランザクションを設定したSessionRepositoryを返す
func (r *SessionRepository) WithTx(tx *sql.Tx) *SessionRepository {
	return &SessionRepository{
		queries: r.queries.WithTx(tx),
	}
}

// GetByToken はトークンでセッションを取得する
func (r *SessionRepository) GetByToken(ctx context.Context, token string) (*model.Session, error) {
	row, err := r.queries.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &model.Session{
		ID:         row.ID,
		ActorID:    row.ActorID,
		Token:      row.Token,
		IPAddress:  row.IpAddress,
		UserAgent:  row.UserAgent,
		SignedInAt: row.SignedInAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

// CreateSessionParams はセッション作成のパラメータ
type CreateSessionParams struct {
	ActorID   uuid.UUID
	Token     string
	IPAddress string
	UserAgent string
}

// Create はセッションを作成する
func (r *SessionRepository) Create(ctx context.Context, params CreateSessionParams) (*model.Session, error) {
	row, err := r.queries.CreateSession(ctx, query.CreateSessionParams{
		ActorID:   params.ActorID,
		Token:     params.Token,
		IpAddress: params.IPAddress,
		UserAgent: params.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	return &model.Session{
		ID:         row.ID,
		ActorID:    row.ActorID,
		Token:      row.Token,
		IPAddress:  row.IpAddress,
		UserAgent:  row.UserAgent,
		SignedInAt: row.SignedInAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

// DeleteByToken はトークンでセッションを削除する
func (r *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	return r.queries.DeleteSessionByToken(ctx, token)
}
