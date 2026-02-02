package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// UserBuilder はユーザーテストデータのビルダー
type UserBuilder struct {
	t              *testing.T
	tx             *sql.Tx
	email          string
	passwordDigest string
	locale         string
	timeZone       string
}

// NewUserBuilder はUserBuilderを生成する
func NewUserBuilder(t *testing.T, tx *sql.Tx) *UserBuilder {
	t.Helper()
	return &UserBuilder{
		t:              t,
		tx:             tx,
		email:          fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
		passwordDigest: "$2a$10$fVAfh.ILhcWBVH1UyokEEedHoNLxozZUTGkoeVnQj9TpZwWPv3ZZS", // "password"
		locale:         "ja",
		timeZone:       "Asia/Tokyo",
	}
}

// WithEmail はメールアドレスを設定する
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.email = email
	return b
}

// WithPasswordDigest はパスワードダイジェストを設定する
func (b *UserBuilder) WithPasswordDigest(passwordDigest string) *UserBuilder {
	b.passwordDigest = passwordDigest
	return b
}

// WithLocale はロケールを設定する
func (b *UserBuilder) WithLocale(locale string) *UserBuilder {
	b.locale = locale
	return b
}

// WithTimeZone はタイムゾーンを設定する
func (b *UserBuilder) WithTimeZone(timeZone string) *UserBuilder {
	b.timeZone = timeZone
	return b
}

// Build はユーザーをDBに作成し、IDを返す
func (b *UserBuilder) Build() uuid.UUID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO users (email, password_digest, locale, time_zone, signed_up_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, b.email, b.passwordDigest, b.locale, b.timeZone, now, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("ユーザーの作成に失敗: %v", err)
	}

	return id
}

// ProfileBuilder はプロフィールテストデータのビルダー
type ProfileBuilder struct {
	t           *testing.T
	tx          *sql.Tx
	ownerType   string
	atname      string
	name        string
	description string
}

// NewProfileBuilder はProfileBuilderを生成する
func NewProfileBuilder(t *testing.T, tx *sql.Tx) *ProfileBuilder {
	t.Helper()
	return &ProfileBuilder{
		t:           t,
		tx:          tx,
		ownerType:   "Actor",
		atname:      fmt.Sprintf("user%d", time.Now().UnixNano()),
		name:        "Test User",
		description: "",
	}
}

// WithAtname は@nameを設定する
func (b *ProfileBuilder) WithAtname(atname string) *ProfileBuilder {
	b.atname = atname
	return b
}

// WithName は表示名を設定する
func (b *ProfileBuilder) WithName(name string) *ProfileBuilder {
	b.name = name
	return b
}

// Build はプロフィールをDBに作成し、IDを返す
func (b *ProfileBuilder) Build() uuid.UUID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO profiles (owner_type, atname, name, description, joined_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, b.ownerType, b.atname, b.name, b.description, now, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("プロフィールの作成に失敗: %v", err)
	}

	return id
}

// ActorBuilder はアクターテストデータのビルダー
type ActorBuilder struct {
	t         *testing.T
	tx        *sql.Tx
	userID    uuid.UUID
	profileID uuid.UUID
}

// NewActorBuilder はActorBuilderを生成する
func NewActorBuilder(t *testing.T, tx *sql.Tx) *ActorBuilder {
	t.Helper()
	return &ActorBuilder{
		t:  t,
		tx: tx,
	}
}

// WithUserID はユーザーIDを設定する
func (b *ActorBuilder) WithUserID(userID uuid.UUID) *ActorBuilder {
	b.userID = userID
	return b
}

// WithProfileID はプロフィールIDを設定する
func (b *ActorBuilder) WithProfileID(profileID uuid.UUID) *ActorBuilder {
	b.profileID = profileID
	return b
}

// Build はアクターをDBに作成し、IDを返す
func (b *ActorBuilder) Build() uuid.UUID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO actors (user_id, profile_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, b.userID, b.profileID, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("アクターの作成に失敗: %v", err)
	}

	return id
}

// SessionBuilder はセッションテストデータのビルダー
type SessionBuilder struct {
	t         *testing.T
	tx        *sql.Tx
	actorID   uuid.UUID
	token     string
	ipAddress string
	userAgent string
}

// NewSessionBuilder はSessionBuilderを生成する
func NewSessionBuilder(t *testing.T, tx *sql.Tx) *SessionBuilder {
	t.Helper()
	return &SessionBuilder{
		t:         t,
		tx:        tx,
		token:     fmt.Sprintf("test-token-%d", time.Now().UnixNano()),
		ipAddress: "127.0.0.1",
		userAgent: "Mozilla/5.0 (Test)",
	}
}

// WithActorID はアクターIDを設定する
func (b *SessionBuilder) WithActorID(actorID uuid.UUID) *SessionBuilder {
	b.actorID = actorID
	return b
}

// WithToken はトークンを設定する
func (b *SessionBuilder) WithToken(token string) *SessionBuilder {
	b.token = token
	return b
}

// WithIPAddress はIPアドレスを設定する
func (b *SessionBuilder) WithIPAddress(ipAddress string) *SessionBuilder {
	b.ipAddress = ipAddress
	return b
}

// WithUserAgent はUser-Agentを設定する
func (b *SessionBuilder) WithUserAgent(userAgent string) *SessionBuilder {
	b.userAgent = userAgent
	return b
}

// Build はセッションをDBに作成し、IDを返す
func (b *SessionBuilder) Build() uuid.UUID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO sessions (actor_id, token, ip_address, user_agent, signed_in_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, b.actorID, b.token, b.ipAddress, b.userAgent, now, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("セッションの作成に失敗: %v", err)
	}

	return id
}

// EmailConfirmationBuilder はメール確認テストデータのビルダー
type EmailConfirmationBuilder struct {
	t           *testing.T
	tx          *sql.Tx
	email       string
	event       string
	code        string
	succeededAt *time.Time
	createdAt   *time.Time
}

// NewEmailConfirmationBuilder はEmailConfirmationBuilderを生成する
func NewEmailConfirmationBuilder(t *testing.T, tx *sql.Tx) *EmailConfirmationBuilder {
	t.Helper()
	return &EmailConfirmationBuilder{
		t:     t,
		tx:    tx,
		email: fmt.Sprintf("test-%d@example.com", time.Now().UnixNano()),
		event: "password_reset",
		code:  "123456",
	}
}

// WithEmail はメールアドレスを設定する
func (b *EmailConfirmationBuilder) WithEmail(email string) *EmailConfirmationBuilder {
	b.email = email
	return b
}

// WithEvent はイベント種別を設定する
func (b *EmailConfirmationBuilder) WithEvent(event string) *EmailConfirmationBuilder {
	b.event = event
	return b
}

// WithCode は確認コードを設定する
func (b *EmailConfirmationBuilder) WithCode(code string) *EmailConfirmationBuilder {
	b.code = code
	return b
}

// WithSucceededAt は成功日時を設定する
func (b *EmailConfirmationBuilder) WithSucceededAt(succeededAt time.Time) *EmailConfirmationBuilder {
	b.succeededAt = &succeededAt
	return b
}

// WithCreatedAt は作成日時を設定する
func (b *EmailConfirmationBuilder) WithCreatedAt(createdAt time.Time) *EmailConfirmationBuilder {
	b.createdAt = &createdAt
	return b
}

// Build はメール確認をDBに作成し、IDを返す
func (b *EmailConfirmationBuilder) Build() uuid.UUID {
	b.t.Helper()

	now := time.Now()
	createdAt := now
	if b.createdAt != nil {
		createdAt = *b.createdAt
	}

	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO email_confirmations (email, event, code, succeeded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, b.email, b.event, b.code, b.succeededAt, createdAt, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("メール確認の作成に失敗: %v", err)
	}

	return id
}
