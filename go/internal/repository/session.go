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
	return toSessionModel(row), nil
}

// AuthLookup bundles the actor / user / profile resolved from a session token
// by FindAuthByToken. All three fields are non-nil when present; the entire
// pointer is nil when the session token is missing or unknown.
//
// [Ja] AuthLookup は FindAuthByToken がトークンから解決した actor / user /
// profile をまとめて保持する。3 つすべてが non-nil。トークンが存在しない、
// または該当セッションが無い場合はポインタ自体が nil となる。
type AuthLookup struct {
	Actor   *model.Actor
	User    *model.User
	Profile *model.Profile
}

// FindAuthByToken resolves a session token to actor / user / profile with a
// single JOIN query, returning them together. This lets the authenticated-page
// middleware resolve the full auth context in one round-trip instead of four
// (session, actor, user, profile). Returns (nil, nil) when the session is
// missing or unknown.
//
// [Ja] FindAuthByToken はトークンから actor / user / profile を 1 度の JOIN
// クエリで解決し、まとめて返す。認証後ページの middleware が認証コンテキスト
// を 1 ラウンドトリップで取得できるようにするためのもので、token -> session
// -> actor -> user / profile の 4 クエリを 1 クエリに圧縮する。トークンが
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
	return toSessionModel(row), nil
}

// DeleteByToken はトークンでセッションを削除する
func (r *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	return r.q.DeleteSessionByToken(ctx, token)
}

// toSessionModel converts a query.Session row into a model.Session. It mirrors
// toActorModel / toUserModel / toProfileModel so all row-to-model converters in
// this package follow the same package-private free-function form.
//
// [Ja] toSessionModel は query.Session を model.Session に変換するパッケージ
// 非公開の自由関数。actor / user / profile の同型の関数と揃え、本パッケージ
// 内の row → model 変換を同じ形式に統一する。
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
