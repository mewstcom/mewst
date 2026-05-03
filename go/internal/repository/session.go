package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// SessionRepository はセッションのリポジトリ
type SessionRepository struct {
	q *query.Queries
}

// NewSessionRepository はSessionRepositoryを生成する
func NewSessionRepository(q *query.Queries) *SessionRepository {
	return &SessionRepository{q: q}
}

// WithTx はトランザクションを設定したSessionRepositoryを返す
func (r *SessionRepository) WithTx(tx *sql.Tx) *SessionRepository {
	return &SessionRepository{q: r.q.WithTx(tx)}
}

// FindByToken はトークンでセッションを取得する
func (r *SessionRepository) FindByToken(ctx context.Context, token string) (*model.Session, error) {
	row, err := r.q.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// CreateSessionInput はセッション作成の入力パラメータ
type CreateSessionInput struct {
	ActorID   model.ActorID
	Token     string
	IPAddress string
	UserAgent string
}

// Create はセッションを作成する
func (r *SessionRepository) Create(ctx context.Context, input CreateSessionInput) (*model.Session, error) {
	row, err := r.q.CreateSession(ctx, query.CreateSessionParams{
		ActorID:   uuid.UUID(input.ActorID),
		Token:     input.Token,
		IpAddress: input.IPAddress,
		UserAgent: input.UserAgent,
	})
	if err != nil {
		return nil, err
	}
	return r.toModel(row), nil
}

// DeleteByToken はトークンでセッションを削除する
func (r *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	return r.q.DeleteSessionByToken(ctx, token)
}

// toModel は query.Session を model.Session に変換する
func (r *SessionRepository) toModel(row query.Session) *model.Session {
	return &model.Session{
		ID:         model.SessionID(row.ID),
		ActorID:    model.ActorID(row.ActorID),
		Token:      row.Token,
		IPAddress:  row.IpAddress,
		UserAgent:  row.UserAgent,
		SignedInAt: row.SignedInAt,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
