package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
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
func (b *UserBuilder) Build() model.UserID {
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

	return model.UserID(id)
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
func (b *ProfileBuilder) Build() model.ProfileID {
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

	return model.ProfileID(id)
}

// ActorBuilder はアクターテストデータのビルダー
type ActorBuilder struct {
	t         *testing.T
	tx        *sql.Tx
	userID    model.UserID
	profileID model.ProfileID
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
func (b *ActorBuilder) WithUserID(userID model.UserID) *ActorBuilder {
	b.userID = userID
	return b
}

// WithProfileID はプロフィールIDを設定する
func (b *ActorBuilder) WithProfileID(profileID model.ProfileID) *ActorBuilder {
	b.profileID = profileID
	return b
}

// Build はアクターをDBに作成し、IDを返す
func (b *ActorBuilder) Build() model.ActorID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO actors (user_id, profile_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, uuid.UUID(b.userID), uuid.UUID(b.profileID), now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("アクターの作成に失敗: %v", err)
	}

	return model.ActorID(id)
}

// SessionBuilder はセッションテストデータのビルダー
type SessionBuilder struct {
	t         *testing.T
	tx        *sql.Tx
	actorID   model.ActorID
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
func (b *SessionBuilder) WithActorID(actorID model.ActorID) *SessionBuilder {
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
func (b *SessionBuilder) Build() model.SessionID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO sessions (actor_id, token, ip_address, user_agent, signed_in_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, uuid.UUID(b.actorID), b.token, b.ipAddress, b.userAgent, now, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("セッションの作成に失敗: %v", err)
	}

	return model.SessionID(id)
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
func (b *EmailConfirmationBuilder) Build() model.EmailConfirmationID {
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

	return model.EmailConfirmationID(id)
}

// FeatureFlagBuilder builds feature flag test data.
// [Ja] FeatureFlagBuilder はフィーチャーフラグテストデータのビルダー。
type FeatureFlagBuilder struct {
	t           *testing.T
	tx          *sql.Tx
	deviceToken string
	actorID     *model.ActorID
	name        string
}

// NewFeatureFlagBuilder creates a FeatureFlagBuilder.
// [Ja] NewFeatureFlagBuilder は FeatureFlagBuilder を生成する。
func NewFeatureFlagBuilder(t *testing.T, tx *sql.Tx) *FeatureFlagBuilder {
	t.Helper()
	return &FeatureFlagBuilder{
		t:    t,
		tx:   tx,
		name: string(model.FeatureFlagExample),
	}
}

// WithDeviceToken sets the device token.
// [Ja] WithDeviceToken はデバイストークンを設定する。
func (b *FeatureFlagBuilder) WithDeviceToken(deviceToken string) *FeatureFlagBuilder {
	b.deviceToken = deviceToken
	return b
}

// WithActorID sets the actor ID.
// [Ja] WithActorID はアクター ID を設定する。
func (b *FeatureFlagBuilder) WithActorID(actorID model.ActorID) *FeatureFlagBuilder {
	b.actorID = &actorID
	return b
}

// WithName sets the feature flag name.
// [Ja] WithName はフィーチャーフラグ名を設定する。
func (b *FeatureFlagBuilder) WithName(name model.FeatureFlagName) *FeatureFlagBuilder {
	b.name = string(name)
	return b
}

// Build inserts the feature flag into the DB and returns its ID.
// [Ja] Build はフィーチャーフラグを DB に作成し、ID を返す。
func (b *FeatureFlagBuilder) Build() model.FeatureFlagID {
	b.t.Helper()

	var deviceToken sql.NullString
	if b.deviceToken != "" {
		deviceToken = sql.NullString{String: b.deviceToken, Valid: true}
	}

	var actorID uuid.NullUUID
	if b.actorID != nil {
		actorID = uuid.NullUUID{UUID: uuid.UUID(*b.actorID), Valid: true}
	}

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO feature_flags (device_token, actor_id, name, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, deviceToken, actorID, b.name, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("フィーチャーフラグの作成に失敗: %v", err)
	}

	return model.FeatureFlagID(id)
}

// OauthApplicationBuilder builds OAuth application test data.
//
// The test DB (reset from schema.sql) has no mewst-web record, so tests that
// attribute posts to mewst-web build one with this builder, setting the uid
// explicitly via WithUID(model.MewstWebUID). The name and uid default to
// per-call unique values so that parallel tests do not contend on the unique
// indexes on oauth_applications.name / .uid.
//
// [Ja] OauthApplicationBuilder は OAuth アプリケーションテストデータのビルダー。
//
// テスト DB (schema.sql からリセット) には mewst-web レコードが無いため、
// 投稿を mewst-web に紐づけるテストは本ビルダーでレコードを作成し、uid は
// WithUID(model.MewstWebUID) で明示的に設定する。name と uid は呼び出しごとに
// ユニークな値をデフォルトにしており、並行テストが oauth_applications.name /
// .uid の unique インデックスで競合しないようにしている。
type OauthApplicationBuilder struct {
	t           *testing.T
	tx          *sql.Tx
	name        string
	uid         string
	secret      string
	redirectURI string
}

// NewOauthApplicationBuilder creates an OauthApplicationBuilder.
// [Ja] NewOauthApplicationBuilder は OauthApplicationBuilder を生成する。
func NewOauthApplicationBuilder(t *testing.T, tx *sql.Tx) *OauthApplicationBuilder {
	t.Helper()
	return &OauthApplicationBuilder{
		t:           t,
		tx:          tx,
		name:        fmt.Sprintf("Test App %d", time.Now().UnixNano()),
		uid:         fmt.Sprintf("test-uid-%d", time.Now().UnixNano()),
		secret:      fmt.Sprintf("test-secret-%d", time.Now().UnixNano()),
		redirectURI: "https://example.com/callback",
	}
}

// WithName sets the application name.
// [Ja] WithName はアプリケーション名を設定する。
func (b *OauthApplicationBuilder) WithName(name string) *OauthApplicationBuilder {
	b.name = name
	return b
}

// WithUID sets the application uid.
// [Ja] WithUID はアプリケーションの uid を設定する。
func (b *OauthApplicationBuilder) WithUID(uid string) *OauthApplicationBuilder {
	b.uid = uid
	return b
}

// Build inserts the OAuth application into the DB and returns its ID.
// [Ja] Build は OAuth アプリケーションを DB に作成し、ID を返す。
func (b *OauthApplicationBuilder) Build() model.OauthApplicationID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO oauth_applications (name, uid, secret, redirect_uri, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, b.name, b.uid, b.secret, b.redirectURI, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("OAuth アプリケーションの作成に失敗: %v", err)
	}

	return model.OauthApplicationID(id)
}

// PostBuilder builds post test data.
// [Ja] PostBuilder は投稿テストデータのビルダー。
type PostBuilder struct {
	t                  *testing.T
	tx                 *sql.Tx
	profileID          model.ProfileID
	oauthApplicationID model.OauthApplicationID
	content            string
	publishedAt        time.Time
}

// NewPostBuilder creates a PostBuilder.
// [Ja] NewPostBuilder は PostBuilder を生成する。
func NewPostBuilder(t *testing.T, tx *sql.Tx) *PostBuilder {
	t.Helper()
	return &PostBuilder{
		t:           t,
		tx:          tx,
		content:     "test post",
		publishedAt: time.Now(),
	}
}

// WithProfileID sets the profile ID.
// [Ja] WithProfileID はプロフィール ID を設定する。
func (b *PostBuilder) WithProfileID(profileID model.ProfileID) *PostBuilder {
	b.profileID = profileID
	return b
}

// WithOauthApplicationID sets the OAuth application ID.
// [Ja] WithOauthApplicationID は OAuth アプリケーション ID を設定する。
func (b *PostBuilder) WithOauthApplicationID(oauthApplicationID model.OauthApplicationID) *PostBuilder {
	b.oauthApplicationID = oauthApplicationID
	return b
}

// WithContent sets the post content.
// [Ja] WithContent は投稿本文を設定する。
func (b *PostBuilder) WithContent(content string) *PostBuilder {
	b.content = content
	return b
}

// WithPublishedAt sets the published time.
// [Ja] WithPublishedAt は公開日時を設定する。
func (b *PostBuilder) WithPublishedAt(publishedAt time.Time) *PostBuilder {
	b.publishedAt = publishedAt
	return b
}

// Build inserts the post into the DB and returns its ID.
// [Ja] Build は投稿を DB に作成し、ID を返す。
func (b *PostBuilder) Build() model.PostID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO posts (profile_id, content, published_at, oauth_application_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, uuid.UUID(b.profileID), b.content, b.publishedAt, uuid.UUID(b.oauthApplicationID), now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("投稿の作成に失敗: %v", err)
	}

	return model.PostID(id)
}

// FollowBuilder builds follow test data.
// [Ja] FollowBuilder はフォローテストデータのビルダー。
type FollowBuilder struct {
	t               *testing.T
	tx              *sql.Tx
	sourceProfileID model.ProfileID
	targetProfileID model.ProfileID
}

// NewFollowBuilder creates a FollowBuilder.
// [Ja] NewFollowBuilder は FollowBuilder を生成する。
func NewFollowBuilder(t *testing.T, tx *sql.Tx) *FollowBuilder {
	t.Helper()
	return &FollowBuilder{
		t:  t,
		tx: tx,
	}
}

// WithSourceProfileID sets the source profile ID (the follower).
// [Ja] WithSourceProfileID は source プロフィール ID (フォローする側) を設定する。
func (b *FollowBuilder) WithSourceProfileID(sourceProfileID model.ProfileID) *FollowBuilder {
	b.sourceProfileID = sourceProfileID
	return b
}

// WithTargetProfileID sets the target profile ID (the followed profile).
// [Ja] WithTargetProfileID は target プロフィール ID (フォローされる側) を設定する。
func (b *FollowBuilder) WithTargetProfileID(targetProfileID model.ProfileID) *FollowBuilder {
	b.targetProfileID = targetProfileID
	return b
}

// Build inserts the follow into the DB and returns its ID.
// [Ja] Build はフォローを DB に作成し、ID を返す。
func (b *FollowBuilder) Build() model.FollowID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO follows (source_profile_id, target_profile_id, followed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, uuid.UUID(b.sourceProfileID), uuid.UUID(b.targetProfileID), now, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("フォローの作成に失敗: %v", err)
	}

	return model.FollowID(id)
}

// LinkBuilder builds link test data. The canonical URL defaults to a per-call
// unique value so that parallel tests do not contend on the unique index on
// links.canonical_url.
//
// [Ja] LinkBuilder はリンクテストデータのビルダー。canonical URL は呼び出しごとに
// ユニークな値をデフォルトにしており、並行テストが links.canonical_url の unique
// インデックスで競合しないようにしている。
type LinkBuilder struct {
	t            *testing.T
	tx           *sql.Tx
	canonicalURL string
	domain       string
	title        string
	imageURL     string
}

// NewLinkBuilder creates a LinkBuilder.
// [Ja] NewLinkBuilder は LinkBuilder を生成する。
func NewLinkBuilder(t *testing.T, tx *sql.Tx) *LinkBuilder {
	t.Helper()
	return &LinkBuilder{
		t:            t,
		tx:           tx,
		canonicalURL: fmt.Sprintf("https://example.com/%d", time.Now().UnixNano()),
		domain:       "example.com",
		title:        "Test Link",
		imageURL:     "https://example.com/image.png",
	}
}

// WithCanonicalURL sets the canonical URL.
// [Ja] WithCanonicalURL は canonical URL を設定する。
func (b *LinkBuilder) WithCanonicalURL(canonicalURL string) *LinkBuilder {
	b.canonicalURL = canonicalURL
	return b
}

// WithDomain sets the domain.
// [Ja] WithDomain はドメインを設定する。
func (b *LinkBuilder) WithDomain(domain string) *LinkBuilder {
	b.domain = domain
	return b
}

// WithTitle sets the title.
// [Ja] WithTitle はタイトルを設定する。
func (b *LinkBuilder) WithTitle(title string) *LinkBuilder {
	b.title = title
	return b
}

// WithImageURL sets the image URL.
// [Ja] WithImageURL は画像 URL を設定する。
func (b *LinkBuilder) WithImageURL(imageURL string) *LinkBuilder {
	b.imageURL = imageURL
	return b
}

// Build inserts the link into the DB and returns its ID.
// [Ja] Build はリンクを DB に作成し、ID を返す。
func (b *LinkBuilder) Build() model.LinkID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO links (canonical_url, domain, title, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, b.canonicalURL, b.domain, b.title, b.imageURL, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("リンクの作成に失敗: %v", err)
	}

	return model.LinkID(id)
}

// PostLinkBuilder builds post-link association test data.
// [Ja] PostLinkBuilder は投稿とリンクの関連付けテストデータのビルダー。
type PostLinkBuilder struct {
	t      *testing.T
	tx     *sql.Tx
	postID model.PostID
	linkID model.LinkID
}

// NewPostLinkBuilder creates a PostLinkBuilder.
// [Ja] NewPostLinkBuilder は PostLinkBuilder を生成する。
func NewPostLinkBuilder(t *testing.T, tx *sql.Tx) *PostLinkBuilder {
	t.Helper()
	return &PostLinkBuilder{
		t:  t,
		tx: tx,
	}
}

// WithPostID sets the post ID.
// [Ja] WithPostID は投稿 ID を設定する。
func (b *PostLinkBuilder) WithPostID(postID model.PostID) *PostLinkBuilder {
	b.postID = postID
	return b
}

// WithLinkID sets the link ID.
// [Ja] WithLinkID はリンク ID を設定する。
func (b *PostLinkBuilder) WithLinkID(linkID model.LinkID) *PostLinkBuilder {
	b.linkID = linkID
	return b
}

// Build inserts the post-link association into the DB and returns its ID.
// [Ja] Build は投稿とリンクの関連付けを DB に作成し、ID を返す。
func (b *PostLinkBuilder) Build() model.PostLinkID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO post_links (post_id, link_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, uuid.UUID(b.postID), uuid.UUID(b.linkID), now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("投稿とリンクの関連付けの作成に失敗: %v", err)
	}

	return model.PostLinkID(id)
}
