package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// SessionRepositoryはセッションのリポジトリ
type SessionRepository struct {
	q *query.Queries
}

// NewSessionRepositoryはSessionRepositoryを生成する
func NewSessionRepository(q *query.Queries) *SessionRepository {
	return &SessionRepository{q: q}
}

// WithTxはトランザクションを設定したSessionRepositoryを返す
func (r *SessionRepository) WithTx(tx *sql.Tx) *SessionRepository {
	return &SessionRepository{q: r.q.WithTx(tx)}
}

// FindByTokenはトークンでセッションを取得する
func (r *SessionRepository) FindByToken(ctx context.Context, token string) (*model.Session, error) {
	row, err := r.q.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return toSessionModel(row), nil
}

// AuthLookupはFindAuthByTokenがトークンから解決したactor / user /
// profileをまとめて保持する。3つすべてがnon-nil。トークンが存在しない、
// または該当セッションが無い場合はポインタ自体がnilとなる。
type AuthLookup struct {
	Actor   *model.Actor
	User    *model.User
	Profile *model.Profile
}

// FindAuthByTokenはトークンからactor / user / profileを1度のJOIN
// クエリで解決し、まとめて返す。認証後ページのmiddlewareが認証コンテキスト
// を1ラウンドトリップで取得できるようにするためのもので、token -> session
// -> actor -> user / profileの4クエリを1クエリに圧縮する。トークンが
// 存在しない、または該当セッションが無い場合は (nil, nil) を返す。
func (r *SessionRepository) FindAuthByToken(ctx context.Context, token string) (*AuthLookup, error) {
	row, err := r.q.GetAuthByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &AuthLookup{
		Actor:   toActorModel(row.Actor),
		User:    toUserModel(row.User),
		Profile: toProfileModel(row.Profile),
	}, nil
}

// CreateSessionInputはセッション作成の入力パラメータ
type CreateSessionInput struct {
	ActorID   model.ActorID
	Token     string
	IPAddress string
	UserAgent string
}

// Createはセッションを作成する
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
	return toSessionModel(row), nil
}

// DeleteByTokenはトークンでセッションを削除する
func (r *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	return r.q.DeleteSessionByToken(ctx, token)
}

// toSessionModelはquery.Sessionをmodel.Sessionに変換するパッケージ
// 非公開の自由関数。actor / user / profileの同型の関数と揃え、本パッケージ
// 内のrow → model変換を同じ形式に統一する。
func toSessionModel(row query.Session) *model.Session {
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
