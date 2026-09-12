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
	t              testing.TB
	tx             *sql.Tx
	email          string
	passwordDigest string
	locale         string
	timeZone       string
}

// NewUserBuilder はUserBuilderを生成する
func NewUserBuilder(t testing.TB, tx *sql.Tx) *UserBuilder {
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
	t           testing.TB
	tx          *sql.Tx
	ownerType   string
	atname      string
	name        string
	description string
	discardedAt sql.NullTime
}

// NewProfileBuilder はProfileBuilderを生成する
func NewProfileBuilder(t testing.TB, tx *sql.Tx) *ProfileBuilder {
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

// WithOwnerType sets the profile owner type.
//
// [Ja] WithOwnerType はプロフィールの所有種別を設定する。
func (b *ProfileBuilder) WithOwnerType(ownerType string) *ProfileBuilder {
	b.ownerType = ownerType
	return b
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

// WithDiscardedAt marks the profile as discarded (soft-deleted) at the given time.
// [Ja] WithDiscardedAt はプロフィールを指定時刻で discard (論理削除) 済みにする。
func (b *ProfileBuilder) WithDiscardedAt(discardedAt time.Time) *ProfileBuilder {
	b.discardedAt = sql.NullTime{Time: discardedAt, Valid: true}
	return b
}

// Build はプロフィールをDBに作成し、IDを返す
func (b *ProfileBuilder) Build() model.ProfileID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO profiles (owner_type, atname, name, description, joined_at, discarded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, b.ownerType, b.atname, b.name, b.description, now, b.discardedAt, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("プロフィールの作成に失敗: %v", err)
	}

	return model.ProfileID(id)
}

// ActorBuilder はアクターテストデータのビルダー
type ActorBuilder struct {
	t         testing.TB
	tx        *sql.Tx
	userID    model.UserID
	profileID model.ProfileID
}

// NewActorBuilder はActorBuilderを生成する
func NewActorBuilder(t testing.TB, tx *sql.Tx) *ActorBuilder {
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

// UserProfileBuilder builds the association that records a user owning a
// profile. Ownership is what authorizes a user to act on a profile's data, so
// tests that go through an authorization check need this row and not only an
// actor.
//
// [Ja] UserProfileBuilder はユーザーがプロフィールを所有していることを記録する
// 関連付けのビルダー。ユーザーがプロフィールのデータを操作できる根拠は所有関係で
// あるため、認可を通るテストは actor だけでなくこの行を必要とする。
type UserProfileBuilder struct {
	t         testing.TB
	tx        *sql.Tx
	userID    model.UserID
	profileID model.ProfileID
}

// NewUserProfileBuilder creates a UserProfileBuilder.
//
// [Ja] NewUserProfileBuilder は UserProfileBuilder を生成する。
func NewUserProfileBuilder(t testing.TB, tx *sql.Tx) *UserProfileBuilder {
	t.Helper()
	return &UserProfileBuilder{
		t:  t,
		tx: tx,
	}
}

// WithUserID sets the owning user ID.
//
// [Ja] WithUserID は所有するユーザー ID を設定する。
func (b *UserProfileBuilder) WithUserID(userID model.UserID) *UserProfileBuilder {
	b.userID = userID
	return b
}

// WithProfileID sets the owned profile ID.
//
// [Ja] WithProfileID は所有されるプロフィール ID を設定する。
func (b *UserProfileBuilder) WithProfileID(profileID model.ProfileID) *UserProfileBuilder {
	b.profileID = profileID
	return b
}

// Build inserts the association into the DB and returns its ID.
//
// [Ja] Build は関連付けを DB に作成し、ID を返す。
func (b *UserProfileBuilder) Build() model.UserProfileID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO user_profiles (user_id, profile_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, uuid.UUID(b.userID), uuid.UUID(b.profileID), now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("ユーザープロフィール関連付けの作成に失敗: %v", err)
	}

	return model.UserProfileID(id)
}

// ProfileOwner is a profile together with the user who owns it and the actor
// that acts as that user on it. The right to act on a profile's data comes
// from the ownership, so a test that goes through an authorization check needs
// all of these rows and not only an actor.
//
// [Ja] ProfileOwner はプロフィールと、それを所有するユーザー、およびその
// ユーザーとしてそのプロフィール上で活動するアクター。プロフィールのデータを
// 操作できる根拠は所有関係であるため、認可を通るテストは actor だけでなく
// これらの行をすべて必要とする。
type ProfileOwner struct {
	UserID    model.UserID
	ProfileID model.ProfileID
	ActorID   model.ActorID
}

// NewProfileOwner creates a user, a profile, the association recording that the
// user owns it, and the actor for that pair.
//
// [Ja] NewProfileOwner はユーザー、プロフィール、そのユーザーがプロフィールを
// 所有していることを記録する関連付け、およびその組み合わせのアクターを作成する。
func NewProfileOwner(t testing.TB, tx *sql.Tx) ProfileOwner {
	t.Helper()

	userID := NewUserBuilder(t, tx).Build()
	profileID := NewProfileBuilder(t, tx).
		WithOwnerType(model.ProfileOwnerTypeUser).
		Build()
	NewUserProfileBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()
	actorID := NewActorBuilder(t, tx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()

	return ProfileOwner{UserID: userID, ProfileID: profileID, ActorID: actorID}
}

// SessionBuilder はセッションテストデータのビルダー
type SessionBuilder struct {
	t         testing.TB
	tx        *sql.Tx
	actorID   model.ActorID
	token     string
	ipAddress string
	userAgent string
}

// NewSessionBuilder はSessionBuilderを生成する
func NewSessionBuilder(t testing.TB, tx *sql.Tx) *SessionBuilder {
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
	t           testing.TB
	tx          *sql.Tx
	email       string
	event       string
	code        string
	succeededAt *time.Time
	createdAt   *time.Time
}

// NewEmailConfirmationBuilder はEmailConfirmationBuilderを生成する
func NewEmailConfirmationBuilder(t testing.TB, tx *sql.Tx) *EmailConfirmationBuilder {
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
	t           testing.TB
	tx          *sql.Tx
	deviceToken string
	actorID     *model.ActorID
	name        string
}

// NewFeatureFlagBuilder creates a FeatureFlagBuilder.
// [Ja] NewFeatureFlagBuilder は FeatureFlagBuilder を生成する。
func NewFeatureFlagBuilder(t testing.TB, tx *sql.Tx) *FeatureFlagBuilder {
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
	t           testing.TB
	tx          *sql.Tx
	name        string
	uid         string
	secret      string
	redirectURI string
}

// NewOauthApplicationBuilder creates an OauthApplicationBuilder.
// [Ja] NewOauthApplicationBuilder は OauthApplicationBuilder を生成する。
func NewOauthApplicationBuilder(t testing.TB, tx *sql.Tx) *OauthApplicationBuilder {
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
	t                  testing.TB
	tx                 *sql.Tx
	profileID          model.ProfileID
	oauthApplicationID model.OauthApplicationID
	content            string
	publishedAt        time.Time
	discardedAt        sql.NullTime
}

// NewPostBuilder creates a PostBuilder.
// [Ja] NewPostBuilder は PostBuilder を生成する。
func NewPostBuilder(t testing.TB, tx *sql.Tx) *PostBuilder {
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

// WithDiscardedAt marks the post as discarded (soft-deleted) at the given time.
// [Ja] WithDiscardedAt は投稿を指定時刻で discard (論理削除) 済みにする。
func (b *PostBuilder) WithDiscardedAt(discardedAt time.Time) *PostBuilder {
	b.discardedAt = sql.NullTime{Time: discardedAt, Valid: true}
	return b
}

// Build inserts the post into the DB and returns its ID.
// [Ja] Build は投稿を DB に作成し、ID を返す。
func (b *PostBuilder) Build() model.PostID {
	b.t.Helper()

	now := time.Now()
	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO posts (profile_id, content, published_at, oauth_application_id, discarded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, uuid.UUID(b.profileID), b.content, b.publishedAt, uuid.UUID(b.oauthApplicationID), b.discardedAt, now, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("投稿の作成に失敗: %v", err)
	}

	return model.PostID(id)
}

// BuildMany inserts count posts in a single statement, spacing their published
// times by interval starting from the builder's published time. It is meant for
// the large fixtures a benchmark needs, where inserting row by row would
// dominate the measurement, so it returns no ID: such a fixture is addressed
// through its profile rather than one post at a time.
//
// [Ja] BuildMany は count 件の投稿を 1 文で挿入し、公開日時をビルダーの公開時刻から
// interval 間隔で並べる。1 行ずつの挿入では計測が挿入時間に支配されるベンチマークの
// 大規模フィクスチャ向けで、ID は返さない。この種のフィクスチャは投稿 1 件ずつでは
// なくプロフィール単位で参照するため。
func (b *PostBuilder) BuildMany(count int, interval time.Duration) {
	b.t.Helper()

	now := time.Now()
	_, err := b.tx.Exec(`
		INSERT INTO posts (profile_id, content, published_at, oauth_application_id, discarded_at, created_at, updated_at)
		SELECT
			$1::uuid,
			$2::text,
			$3::timestamp + ((i - 1) * $4::bigint) * INTERVAL '1 microsecond',
			$5::uuid,
			$6::timestamp,
			$7::timestamp,
			$7::timestamp
		FROM generate_series(1, $8::bigint) AS i
	`, uuid.UUID(b.profileID), b.content, b.publishedAt, interval.Microseconds(), uuid.UUID(b.oauthApplicationID), b.discardedAt, now, count)

	if err != nil {
		b.t.Fatalf("投稿の一括作成に失敗: %v", err)
	}
}

// FollowBuilder builds follow test data.
// [Ja] FollowBuilder はフォローテストデータのビルダー。
type FollowBuilder struct {
	t               testing.TB
	tx              *sql.Tx
	sourceProfileID model.ProfileID
	targetProfileID model.ProfileID
}

// NewFollowBuilder creates a FollowBuilder.
// [Ja] NewFollowBuilder は FollowBuilder を生成する。
func NewFollowBuilder(t testing.TB, tx *sql.Tx) *FollowBuilder {
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
	t            testing.TB
	tx           *sql.Tx
	canonicalURL string
	domain       string
	title        string
	imageURL     string
}

// NewLinkBuilder creates a LinkBuilder.
// [Ja] NewLinkBuilder は LinkBuilder を生成する。
func NewLinkBuilder(t testing.TB, tx *sql.Tx) *LinkBuilder {
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

// ExportBuilder builds export test data. Build derives the state fields
// (object_key / started_at / finished_at) from the status so the inserted row
// always satisfies the exports_state_fields_check constraint.
//
// [Ja] ExportBuilder はエクスポートテストデータのビルダー。Build は status から
// 状態カラム (object_key / started_at / finished_at) を導出し、挿入する行が常に
// exports_state_fields_check 制約を満たすようにする。
type ExportBuilder struct {
	t            testing.TB
	tx           *sql.Tx
	profileID    model.ProfileID
	actorID      model.ActorID
	status       model.ExportStatus
	objectKey    string
	attemptCount int32
	createdAt    *time.Time
}

// NewExportBuilder creates an ExportBuilder. The status defaults to queued.
//
// [Ja] NewExportBuilder は ExportBuilder を生成する。status は queued を既定とする。
func NewExportBuilder(t testing.TB, tx *sql.Tx) *ExportBuilder {
	t.Helper()
	return &ExportBuilder{
		t:      t,
		tx:     tx,
		status: model.ExportStatusQueued,
	}
}

// WithProfileID sets the profile ID (the export target).
//
// [Ja] WithProfileID はプロフィール ID (エクスポート対象) を設定する。
func (b *ExportBuilder) WithProfileID(profileID model.ProfileID) *ExportBuilder {
	b.profileID = profileID
	return b
}

// WithActorID sets the actor ID (the requester).
//
// [Ja] WithActorID はアクター ID (申請者) を設定する。
func (b *ExportBuilder) WithActorID(actorID model.ActorID) *ExportBuilder {
	b.actorID = actorID
	return b
}

// WithStatus sets the export status.
//
// [Ja] WithStatus はエクスポートの status を設定する。
func (b *ExportBuilder) WithStatus(status model.ExportStatus) *ExportBuilder {
	b.status = status
	return b
}

// WithObjectKey sets the object key used for a succeeded export.
//
// [Ja] WithObjectKey は succeeded エクスポートの object key を設定する。
func (b *ExportBuilder) WithObjectKey(objectKey string) *ExportBuilder {
	b.objectKey = objectKey
	return b
}

// WithAttemptCount sets attempt_count, which counts how many times a Worker has
// started the export. Recovery decides between retrying a stalled export and
// giving up on it by this number, and reaching the limit through the Repository
// would also stamp started_at with the current time, leaving the export too
// recent to be recovered.
//
// [Ja] WithAttemptCount は attempt_count (Worker がそのエクスポートの処理を開始した
// 回数) を設定する。回復処理は停滞したエクスポートを再試行するか諦めるかをこの回数で
// 判断する。Repository 経由で上限まで増やすと started_at も現在時刻で打刻され、
// 回復対象になるほど古くないエクスポートになってしまう。
func (b *ExportBuilder) WithAttemptCount(attemptCount int32) *ExportBuilder {
	b.attemptCount = attemptCount
	return b
}

// WithCreatedAt sets the created_at timestamp, letting tests order exports
// deterministically instead of relying on generated IDs.
//
// [Ja] WithCreatedAt は created_at を設定し、生成 ID に頼らずテストが
// エクスポートの順序を決定的に並べられるようにする。
func (b *ExportBuilder) WithCreatedAt(createdAt time.Time) *ExportBuilder {
	b.createdAt = &createdAt
	return b
}

// Build inserts the export into the DB and returns its ID.
//
// [Ja] Build はエクスポートを DB に作成し、ID を返す。
func (b *ExportBuilder) Build() model.ExportID {
	b.t.Helper()

	now := time.Now()
	createdAt := now
	if b.createdAt != nil {
		createdAt = *b.createdAt
	}

	var (
		objectKey  sql.NullString
		startedAt  sql.NullTime
		finishedAt sql.NullTime
	)
	switch b.status {
	case model.ExportStatusStarted:
		startedAt = sql.NullTime{Time: createdAt, Valid: true}
	case model.ExportStatusSucceeded:
		key := b.objectKey
		if key == "" {
			key = fmt.Sprintf("exports/%s/%d.zip", b.profileID, time.Now().UnixNano())
		}
		objectKey = sql.NullString{String: key, Valid: true}
		startedAt = sql.NullTime{Time: createdAt, Valid: true}
		finishedAt = sql.NullTime{Time: createdAt, Valid: true}
	case model.ExportStatusFailed:
		startedAt = sql.NullTime{Time: createdAt, Valid: true}
		finishedAt = sql.NullTime{Time: createdAt, Valid: true}
	}

	var id uuid.UUID
	err := b.tx.QueryRow(`
		INSERT INTO exports (
			profile_id, actor_id, status, object_key, attempt_count, started_at, finished_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, uuid.UUID(b.profileID), uuid.UUID(b.actorID), string(b.status),
		objectKey, b.attemptCount, startedAt, finishedAt, createdAt, now).Scan(&id)

	if err != nil {
		b.t.Fatalf("エクスポートの作成に失敗: %v", err)
	}

	return model.ExportID(id)
}

// ExportCompletionNotificationBuilder builds pending completion-notification
// test data independently of an export row.
//
// [Ja] ExportCompletionNotificationBuilder は export 行から独立した送信待ち完了通知の
// テストデータを作成する。
type ExportCompletionNotificationBuilder struct {
	t              testing.TB
	tx             *sql.Tx
	exportID       model.ExportID
	actorID        model.ActorID
	recipientEmail string
	locale         string
	createdAt      *time.Time
}

// NewExportCompletionNotificationBuilder creates an
// ExportCompletionNotificationBuilder.
//
// [Ja] NewExportCompletionNotificationBuilder は
// ExportCompletionNotificationBuilder を生成する。
func NewExportCompletionNotificationBuilder(t testing.TB, tx *sql.Tx) *ExportCompletionNotificationBuilder {
	t.Helper()
	return &ExportCompletionNotificationBuilder{
		t:              t,
		tx:             tx,
		recipientEmail: fmt.Sprintf("export-%d@example.com", time.Now().UnixNano()),
		locale:         "ja",
	}
}

// WithExportID sets the export ID. The export row does not need to exist.
//
// [Ja] WithExportID は export ID を設定する。export 行が存在する必要はない。
func (b *ExportCompletionNotificationBuilder) WithExportID(exportID model.ExportID) *ExportCompletionNotificationBuilder {
	b.exportID = exportID
	return b
}

// WithActorID sets the requester actor. Deleting it cancels the notification.
//
// [Ja] WithActorID は申請 actor を設定する。actor が削除されると通知も取り消される。
func (b *ExportCompletionNotificationBuilder) WithActorID(actorID model.ActorID) *ExportCompletionNotificationBuilder {
	b.actorID = actorID
	return b
}

// WithRecipientEmail sets the snapshotted recipient email.
//
// [Ja] WithRecipientEmail は snapshot 済みの宛先メールアドレスを設定する。
func (b *ExportCompletionNotificationBuilder) WithRecipientEmail(recipientEmail string) *ExportCompletionNotificationBuilder {
	b.recipientEmail = recipientEmail
	return b
}

// WithLocale sets the snapshotted recipient locale.
//
// [Ja] WithLocale は snapshot 済みの宛先 locale を設定する。
func (b *ExportCompletionNotificationBuilder) WithLocale(locale string) *ExportCompletionNotificationBuilder {
	b.locale = locale
	return b
}

// WithCreatedAt sets when the notification became pending.
//
// [Ja] WithCreatedAt は通知が pending になった時刻を設定する。
func (b *ExportCompletionNotificationBuilder) WithCreatedAt(createdAt time.Time) *ExportCompletionNotificationBuilder {
	b.createdAt = &createdAt
	return b
}

// Build inserts the pending notification. The profile is read from the
// requester the same way the succeeded transition snapshots it, so the pair the
// foreign key checks can never be built inconsistently from a test.
//
// [Ja] Build は送信待ち通知を作成する。プロフィールは succeeded 遷移が snapshot する
// のと同じく申請者から読むため、外部キーが検査する組をテストから不整合に作ることは
// ない。
func (b *ExportCompletionNotificationBuilder) Build() {
	b.t.Helper()

	createdAt := time.Now()
	if b.createdAt != nil {
		createdAt = *b.createdAt
	}

	result, err := b.tx.Exec(`
		INSERT INTO export_completion_notifications (
			export_id, actor_id, profile_id, recipient_email, locale, created_at
		)
		SELECT $1, actors.id, actors.profile_id, $3, $4, $5
		FROM actors
		WHERE actors.id = $2
	`, uuid.UUID(b.exportID), uuid.UUID(b.actorID), b.recipientEmail, b.locale, createdAt)
	if err != nil {
		b.t.Fatalf("エクスポート完了通知の作成に失敗: %v", err)
	}

	inserted, err := result.RowsAffected()
	if err != nil {
		b.t.Fatalf("エクスポート完了通知の作成行数の取得に失敗: %v", err)
	}
	if inserted != 1 {
		b.t.Fatalf("エクスポート完了通知を作成できない (actor_id: %s が存在しない)", b.actorID.String())
	}
}

// PostLinkBuilder builds post-link association test data.
// [Ja] PostLinkBuilder は投稿とリンクの関連付けテストデータのビルダー。
type PostLinkBuilder struct {
	t      testing.TB
	tx     *sql.Tx
	postID model.PostID
	linkID model.LinkID
}

// NewPostLinkBuilder creates a PostLinkBuilder.
// [Ja] NewPostLinkBuilder は PostLinkBuilder を生成する。
func NewPostLinkBuilder(t testing.TB, tx *sql.Tx) *PostLinkBuilder {
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
