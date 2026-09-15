package testutil

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

// UserBuilderはユーザーテストデータのビルダー
type UserBuilder struct {
	t              testing.TB
	tx             *sql.Tx
	email          string
	passwordDigest string
	locale         string
	timeZone       string
}

// NewUserBuilderはUserBuilderを生成する
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

// WithEmailはメールアドレスを設定する
func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.email = email
	return b
}

// WithPasswordDigestはパスワードダイジェストを設定する
func (b *UserBuilder) WithPasswordDigest(passwordDigest string) *UserBuilder {
	b.passwordDigest = passwordDigest
	return b
}

// WithLocaleはロケールを設定する
func (b *UserBuilder) WithLocale(locale string) *UserBuilder {
	b.locale = locale
	return b
}

// WithTimeZoneはタイムゾーンを設定する
func (b *UserBuilder) WithTimeZone(timeZone string) *UserBuilder {
	b.timeZone = timeZone
	return b
}

// BuildはユーザーをDBに作成し、IDを返す
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

// ProfileBuilderはプロフィールテストデータのビルダー
type ProfileBuilder struct {
	t           testing.TB
	tx          *sql.Tx
	ownerType   string
	atname      string
	name        string
	description string
	discardedAt sql.NullTime
}

// NewProfileBuilderはProfileBuilderを生成する
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

// WithOwnerTypeはプロフィールの所有種別を設定する。
func (b *ProfileBuilder) WithOwnerType(ownerType string) *ProfileBuilder {
	b.ownerType = ownerType
	return b
}

// WithAtnameは@nameを設定する
func (b *ProfileBuilder) WithAtname(atname string) *ProfileBuilder {
	b.atname = atname
	return b
}

// WithNameは表示名を設定する
func (b *ProfileBuilder) WithName(name string) *ProfileBuilder {
	b.name = name
	return b
}

// WithDiscardedAtはプロフィールを指定時刻でdiscard (論理削除) 済みにする。
func (b *ProfileBuilder) WithDiscardedAt(discardedAt time.Time) *ProfileBuilder {
	b.discardedAt = sql.NullTime{Time: discardedAt, Valid: true}
	return b
}

// BuildはプロフィールをDBに作成し、IDを返す
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

// ActorBuilderはアクターテストデータのビルダー
type ActorBuilder struct {
	t         testing.TB
	tx        *sql.Tx
	userID    model.UserID
	profileID model.ProfileID
}

// NewActorBuilderはActorBuilderを生成する
func NewActorBuilder(t testing.TB, tx *sql.Tx) *ActorBuilder {
	t.Helper()
	return &ActorBuilder{
		t:  t,
		tx: tx,
	}
}

// WithUserIDはユーザーIDを設定する
func (b *ActorBuilder) WithUserID(userID model.UserID) *ActorBuilder {
	b.userID = userID
	return b
}

// WithProfileIDはプロフィールIDを設定する
func (b *ActorBuilder) WithProfileID(profileID model.ProfileID) *ActorBuilder {
	b.profileID = profileID
	return b
}

// BuildはアクターをDBに作成し、IDを返す
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

// UserProfileBuilderはユーザーがプロフィールを所有していることを記録する
// 関連付けのビルダー。ユーザーがプロフィールのデータを操作できる根拠は所有関係で
// あるため、認可を通るテストはactorだけでなくこの行を必要とする。
type UserProfileBuilder struct {
	t         testing.TB
	tx        *sql.Tx
	userID    model.UserID
	profileID model.ProfileID
}

// NewUserProfileBuilderはUserProfileBuilderを生成する。
func NewUserProfileBuilder(t testing.TB, tx *sql.Tx) *UserProfileBuilder {
	t.Helper()
	return &UserProfileBuilder{
		t:  t,
		tx: tx,
	}
}

// WithUserIDは所有するユーザーIDを設定する。
func (b *UserProfileBuilder) WithUserID(userID model.UserID) *UserProfileBuilder {
	b.userID = userID
	return b
}

// WithProfileIDは所有されるプロフィールIDを設定する。
func (b *UserProfileBuilder) WithProfileID(profileID model.ProfileID) *UserProfileBuilder {
	b.profileID = profileID
	return b
}

// Buildは関連付けをDBに作成し、IDを返す。
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

// ProfileOwnerはプロフィールと、それを所有するユーザー、およびその
// ユーザーとしてそのプロフィール上で活動するアクター。プロフィールのデータを
// 操作できる根拠は所有関係であるため、認可を通るテストはactorだけでなく
// これらの行をすべて必要とする。
type ProfileOwner struct {
	UserID    model.UserID
	ProfileID model.ProfileID
	ActorID   model.ActorID
}

// NewProfileOwnerはユーザー、プロフィール、そのユーザーがプロフィールを
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

// SessionBuilderはセッションテストデータのビルダー
type SessionBuilder struct {
	t         testing.TB
	tx        *sql.Tx
	actorID   model.ActorID
	token     string
	ipAddress string
	userAgent string
}

// NewSessionBuilderはSessionBuilderを生成する
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

// WithActorIDはアクターIDを設定する
func (b *SessionBuilder) WithActorID(actorID model.ActorID) *SessionBuilder {
	b.actorID = actorID
	return b
}

// WithTokenはトークンを設定する
func (b *SessionBuilder) WithToken(token string) *SessionBuilder {
	b.token = token
	return b
}

// WithIPAddressはIPアドレスを設定する
func (b *SessionBuilder) WithIPAddress(ipAddress string) *SessionBuilder {
	b.ipAddress = ipAddress
	return b
}

// WithUserAgentはUser-Agentを設定する
func (b *SessionBuilder) WithUserAgent(userAgent string) *SessionBuilder {
	b.userAgent = userAgent
	return b
}

// BuildはセッションをDBに作成し、IDを返す
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

// EmailConfirmationBuilderはメール確認テストデータのビルダー
type EmailConfirmationBuilder struct {
	t           testing.TB
	tx          *sql.Tx
	email       string
	event       string
	code        string
	succeededAt *time.Time
	createdAt   *time.Time
}

// NewEmailConfirmationBuilderはEmailConfirmationBuilderを生成する
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

// WithEmailはメールアドレスを設定する
func (b *EmailConfirmationBuilder) WithEmail(email string) *EmailConfirmationBuilder {
	b.email = email
	return b
}

// WithEventはイベント種別を設定する
func (b *EmailConfirmationBuilder) WithEvent(event string) *EmailConfirmationBuilder {
	b.event = event
	return b
}

// WithCodeは確認コードを設定する
func (b *EmailConfirmationBuilder) WithCode(code string) *EmailConfirmationBuilder {
	b.code = code
	return b
}

// WithSucceededAtは成功日時を設定する
func (b *EmailConfirmationBuilder) WithSucceededAt(succeededAt time.Time) *EmailConfirmationBuilder {
	b.succeededAt = &succeededAt
	return b
}

// WithCreatedAtは作成日時を設定する
func (b *EmailConfirmationBuilder) WithCreatedAt(createdAt time.Time) *EmailConfirmationBuilder {
	b.createdAt = &createdAt
	return b
}

// Buildはメール確認をDBに作成し、IDを返す
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

// FeatureFlagBuilderはフィーチャーフラグテストデータのビルダー。
type FeatureFlagBuilder struct {
	t           testing.TB
	tx          *sql.Tx
	deviceToken string
	actorID     *model.ActorID
	name        string
}

// NewFeatureFlagBuilderはFeatureFlagBuilderを生成する。
func NewFeatureFlagBuilder(t testing.TB, tx *sql.Tx) *FeatureFlagBuilder {
	t.Helper()
	return &FeatureFlagBuilder{
		t:    t,
		tx:   tx,
		name: string(model.FeatureFlagExample),
	}
}

// WithDeviceTokenはデバイストークンを設定する。
func (b *FeatureFlagBuilder) WithDeviceToken(deviceToken string) *FeatureFlagBuilder {
	b.deviceToken = deviceToken
	return b
}

// WithActorIDはアクターIDを設定する。
func (b *FeatureFlagBuilder) WithActorID(actorID model.ActorID) *FeatureFlagBuilder {
	b.actorID = &actorID
	return b
}

// WithNameはフィーチャーフラグ名を設定する。
func (b *FeatureFlagBuilder) WithName(name model.FeatureFlagName) *FeatureFlagBuilder {
	b.name = string(name)
	return b
}

// BuildはフィーチャーフラグをDBに作成し、IDを返す。
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

// OauthApplicationBuilderはOAuthアプリケーションテストデータのビルダー。
//
// テストDB (schema.sqlからリセット) にはmewst-webレコードが無いため、
// 投稿をmewst-webに紐づけるテストは本ビルダーでレコードを作成し、uidは
// WithUID(model.MewstWebUID) で明示的に設定する。nameとuidは呼び出しごとに
// ユニークな値をデフォルトにしており、並行テストがoauth_applications.name /
// .uidのuniqueインデックスで競合しないようにしている。
type OauthApplicationBuilder struct {
	t           testing.TB
	tx          *sql.Tx
	name        string
	uid         string
	secret      string
	redirectURI string
}

// NewOauthApplicationBuilderはOauthApplicationBuilderを生成する。
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

// WithNameはアプリケーション名を設定する。
func (b *OauthApplicationBuilder) WithName(name string) *OauthApplicationBuilder {
	b.name = name
	return b
}

// WithUIDはアプリケーションのuidを設定する。
func (b *OauthApplicationBuilder) WithUID(uid string) *OauthApplicationBuilder {
	b.uid = uid
	return b
}

// BuildはOAuthアプリケーションをDBに作成し、IDを返す。
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
		b.t.Fatalf("OAuthアプリケーションの作成に失敗: %v", err)
	}

	return model.OauthApplicationID(id)
}

// PostBuilderは投稿テストデータのビルダー。
type PostBuilder struct {
	t                  testing.TB
	tx                 *sql.Tx
	profileID          model.ProfileID
	oauthApplicationID model.OauthApplicationID
	content            string
	publishedAt        time.Time
	discardedAt        sql.NullTime
}

// NewPostBuilderはPostBuilderを生成する。
func NewPostBuilder(t testing.TB, tx *sql.Tx) *PostBuilder {
	t.Helper()
	return &PostBuilder{
		t:           t,
		tx:          tx,
		content:     "test post",
		publishedAt: time.Now(),
	}
}

// WithProfileIDはプロフィールIDを設定する。
func (b *PostBuilder) WithProfileID(profileID model.ProfileID) *PostBuilder {
	b.profileID = profileID
	return b
}

// WithOauthApplicationIDはOAuthアプリケーションIDを設定する。
func (b *PostBuilder) WithOauthApplicationID(oauthApplicationID model.OauthApplicationID) *PostBuilder {
	b.oauthApplicationID = oauthApplicationID
	return b
}

// WithContentは投稿本文を設定する。
func (b *PostBuilder) WithContent(content string) *PostBuilder {
	b.content = content
	return b
}

// WithPublishedAtは公開日時を設定する。
func (b *PostBuilder) WithPublishedAt(publishedAt time.Time) *PostBuilder {
	b.publishedAt = publishedAt
	return b
}

// WithDiscardedAtは投稿を指定時刻でdiscard (論理削除) 済みにする。
func (b *PostBuilder) WithDiscardedAt(discardedAt time.Time) *PostBuilder {
	b.discardedAt = sql.NullTime{Time: discardedAt, Valid: true}
	return b
}

// Buildは投稿をDBに作成し、IDを返す。
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

// BuildManyはcount件の投稿を1文で挿入し、公開日時をビルダーの公開時刻から
// interval間隔で並べる。1行ずつの挿入では計測が挿入時間に支配されるベンチマークの
// 大規模フィクスチャ向けで、IDは返さない。この種のフィクスチャは投稿1件ずつでは
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

// FollowBuilderはフォローテストデータのビルダー。
type FollowBuilder struct {
	t               testing.TB
	tx              *sql.Tx
	sourceProfileID model.ProfileID
	targetProfileID model.ProfileID
}

// NewFollowBuilderはFollowBuilderを生成する。
func NewFollowBuilder(t testing.TB, tx *sql.Tx) *FollowBuilder {
	t.Helper()
	return &FollowBuilder{
		t:  t,
		tx: tx,
	}
}

// WithSourceProfileIDはsourceプロフィールID (フォローする側) を設定する。
func (b *FollowBuilder) WithSourceProfileID(sourceProfileID model.ProfileID) *FollowBuilder {
	b.sourceProfileID = sourceProfileID
	return b
}

// WithTargetProfileIDはtargetプロフィールID (フォローされる側) を設定する。
func (b *FollowBuilder) WithTargetProfileID(targetProfileID model.ProfileID) *FollowBuilder {
	b.targetProfileID = targetProfileID
	return b
}

// BuildはフォローをDBに作成し、IDを返す。
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

// LinkBuilderはリンクテストデータのビルダー。canonical URLは呼び出しごとに
// ユニークな値をデフォルトにしており、並行テストがlinks.canonical_urlのunique
// インデックスで競合しないようにしている。
type LinkBuilder struct {
	t            testing.TB
	tx           *sql.Tx
	canonicalURL string
	domain       string
	title        string
	imageURL     string
}

// NewLinkBuilderはLinkBuilderを生成する。
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

// WithCanonicalURLはcanonical URLを設定する。
func (b *LinkBuilder) WithCanonicalURL(canonicalURL string) *LinkBuilder {
	b.canonicalURL = canonicalURL
	return b
}

// WithDomainはドメインを設定する。
func (b *LinkBuilder) WithDomain(domain string) *LinkBuilder {
	b.domain = domain
	return b
}

// WithTitleはタイトルを設定する。
func (b *LinkBuilder) WithTitle(title string) *LinkBuilder {
	b.title = title
	return b
}

// WithImageURLは画像URLを設定する。
func (b *LinkBuilder) WithImageURL(imageURL string) *LinkBuilder {
	b.imageURL = imageURL
	return b
}

// BuildはリンクをDBに作成し、IDを返す。
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

// ExportBuilderはエクスポートテストデータのビルダー。Buildはstatusから
// 状態カラム (object_key / started_at / finished_at) を導出し、挿入する行が常に
// exports_state_fields_check制約を満たすようにする。
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

// NewExportBuilderはExportBuilderを生成する。statusはqueuedを既定とする。
func NewExportBuilder(t testing.TB, tx *sql.Tx) *ExportBuilder {
	t.Helper()
	return &ExportBuilder{
		t:      t,
		tx:     tx,
		status: model.ExportStatusQueued,
	}
}

// WithProfileIDはプロフィールID (エクスポート対象) を設定する。
func (b *ExportBuilder) WithProfileID(profileID model.ProfileID) *ExportBuilder {
	b.profileID = profileID
	return b
}

// WithActorIDはアクターID (申請者) を設定する。
func (b *ExportBuilder) WithActorID(actorID model.ActorID) *ExportBuilder {
	b.actorID = actorID
	return b
}

// WithStatusはエクスポートのstatusを設定する。
func (b *ExportBuilder) WithStatus(status model.ExportStatus) *ExportBuilder {
	b.status = status
	return b
}

// WithObjectKeyはsucceededエクスポートのobject keyを設定する。
func (b *ExportBuilder) WithObjectKey(objectKey string) *ExportBuilder {
	b.objectKey = objectKey
	return b
}

// WithAttemptCountはattempt_count (Workerがそのエクスポートの処理を開始した
// 回数) を設定する。回復処理は停滞したエクスポートを再試行するか諦めるかをこの回数で
// 判断する。Repository経由で上限まで増やすとstarted_atも現在時刻で打刻され、
// 回復対象になるほど古くないエクスポートになってしまう。
func (b *ExportBuilder) WithAttemptCount(attemptCount int32) *ExportBuilder {
	b.attemptCount = attemptCount
	return b
}

// WithCreatedAtはcreated_atを設定し、生成IDに頼らずテストが
// エクスポートの順序を決定的に並べられるようにする。
func (b *ExportBuilder) WithCreatedAt(createdAt time.Time) *ExportBuilder {
	b.createdAt = &createdAt
	return b
}

// BuildはエクスポートをDBに作成し、IDを返す。
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

// ExportCompletionNotificationBuilderはexport行から独立した送信待ち完了通知の
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

// NewExportCompletionNotificationBuilderは
// ExportCompletionNotificationBuilderを生成する。
func NewExportCompletionNotificationBuilder(t testing.TB, tx *sql.Tx) *ExportCompletionNotificationBuilder {
	t.Helper()
	return &ExportCompletionNotificationBuilder{
		t:              t,
		tx:             tx,
		recipientEmail: fmt.Sprintf("export-%d@example.com", time.Now().UnixNano()),
		locale:         "ja",
	}
}

// WithExportIDはexport IDを設定する。export行が存在する必要はない。
func (b *ExportCompletionNotificationBuilder) WithExportID(exportID model.ExportID) *ExportCompletionNotificationBuilder {
	b.exportID = exportID
	return b
}

// WithActorIDは申請actorを設定する。actorが削除されると通知も取り消される。
func (b *ExportCompletionNotificationBuilder) WithActorID(actorID model.ActorID) *ExportCompletionNotificationBuilder {
	b.actorID = actorID
	return b
}

// WithRecipientEmailはsnapshot済みの宛先メールアドレスを設定する。
func (b *ExportCompletionNotificationBuilder) WithRecipientEmail(recipientEmail string) *ExportCompletionNotificationBuilder {
	b.recipientEmail = recipientEmail
	return b
}

// WithLocaleはsnapshot済みの宛先localeを設定する。
func (b *ExportCompletionNotificationBuilder) WithLocale(locale string) *ExportCompletionNotificationBuilder {
	b.locale = locale
	return b
}

// WithCreatedAtは通知がpendingになった時刻を設定する。
func (b *ExportCompletionNotificationBuilder) WithCreatedAt(createdAt time.Time) *ExportCompletionNotificationBuilder {
	b.createdAt = &createdAt
	return b
}

// Buildは送信待ち通知を作成する。プロフィールはsucceeded遷移がsnapshotする
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
		b.t.Fatalf("エクスポート完了通知を作成できない (actor_id: %sが存在しない)", b.actorID.String())
	}
}

// PostLinkBuilderは投稿とリンクの関連付けテストデータのビルダー。
type PostLinkBuilder struct {
	t      testing.TB
	tx     *sql.Tx
	postID model.PostID
	linkID model.LinkID
}

// NewPostLinkBuilderはPostLinkBuilderを生成する。
func NewPostLinkBuilder(t testing.TB, tx *sql.Tx) *PostLinkBuilder {
	t.Helper()
	return &PostLinkBuilder{
		t:  t,
		tx: tx,
	}
}

// WithPostIDは投稿IDを設定する。
func (b *PostLinkBuilder) WithPostID(postID model.PostID) *PostLinkBuilder {
	b.postID = postID
	return b
}

// WithLinkIDはリンクIDを設定する。
func (b *PostLinkBuilder) WithLinkID(linkID model.LinkID) *PostLinkBuilder {
	b.linkID = linkID
	return b
}

// Buildは投稿とリンクの関連付けをDBに作成し、IDを返す。
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
