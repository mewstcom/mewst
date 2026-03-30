# Go への移行 (サインアップ機能編) 設計書

## 実装ガイドラインの参照

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の8種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 概要

Go 版 Mewst にサインアップ（新規ユーザー登録）機能を実装します。ユーザーはメールアドレスを入力し、確認コードによる本人確認を経て、アットネームとパスワードを設定してアカウントを作成できます。

Rails 版 Mewst と同じ DB を共有するため、Go 版で作成されたアカウントは Rails 版でも利用可能です。

**目的**:

- Rails 版から Go 版への段階的移行の一環として、サインアップ機能を Go 版に移植する
- ユーザーが Go 版で新規アカウントを作成できるようにする
- Rails 版と同等のセキュリティ水準を維持しつつ、Bot 対策（Turnstile）を追加する

**背景**:

- ログイン機能は既に Go 版に移植済み（`@docs/designs/3_done/202601/go.md` 参照）
- サインアップ機能は「スコープ外」として実装を見送っていた
- 新規ユーザー獲得のため、Go 版でもサインアップ機能が必要

## 要件

### 機能要件

- ユーザーはメールアドレスを入力してサインアップを開始できる
- システムは入力されたメールアドレスに確認コード（6桁の数字）を送信する
- ユーザーは確認コードを入力してメールアドレスの所有を証明できる
- 確認コードの有効期限は15分とする
- メール確認完了後、ユーザーはアットネームとパスワードを設定してアカウントを作成できる
- アカウント作成後、自動的にログイン状態になる
- ログイン済みユーザーがサインアップページにアクセスした場合、ホームページにリダイレクトする
- サインアップ完了後にフラッシュメッセージを表示する

### 非機能要件

#### セキュリティ

- パスワードは bcrypt でハッシュ化して保存する
- CSRF 対策を実施する（全フォームに CSRF トークンを含める）
- Cookie には `httponly=true`, `same_site=lax` を設定
- Cloudflare Turnstile による Bot 対策を実施（Rails 版にはなかった新機能）
- メールアドレスの重複チェックを行う
- 確認コードは暗号学的に安全な乱数で生成する
- PostgreSQL ベースのレート制限を実施（Wikino の実装を参考）

#### 国際化

- 日本語と英語の両言語に対応
- エラーメッセージ、フラッシュメッセージ、フォームラベルを国際化

#### Rails 互換性

- Rails 版と同じ `users`, `profiles`, `actors`, `user_profiles` テーブルを使用
- Rails 版と同じ `email_confirmations` テーブルを使用
- Rails 版で作成されたアカウントと Go 版で作成されたアカウントは同等に扱われる

## 設計

### 技術スタック

- **パスワードハッシュ化**: `golang.org/x/crypto/bcrypt`
- **確認コード生成**: `crypto/rand`
- **セッショントークン生成**: `crypto/rand` + `encoding/base64`
- **HTTP ルーター**: `chi/v5`
- **テンプレート**: `templ`
- **DB アクセス**: `sqlc`
- **Bot 対策**: Cloudflare Turnstile
- **メール送信**: Resend API

### データベース設計

既存の Rails 版テーブルをそのまま使用します。

#### users テーブル（既存）

```sql
CREATE TABLE public.users (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email public.citext NOT NULL,
    password_digest character varying NOT NULL,
    locale character varying NOT NULL,
    signed_up_at timestamp without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    time_zone character varying DEFAULT 'Etc/UTC'::character varying NOT NULL
);
```

#### profiles テーブル（既存）

```sql
CREATE TABLE public.profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    owner_type character varying NOT NULL,
    atname public.citext NOT NULL,
    name character varying DEFAULT ''::character varying NOT NULL,
    description character varying DEFAULT ''::character varying NOT NULL,
    image_url character varying DEFAULT ''::character varying NOT NULL,
    joined_at timestamp(6) without time zone NOT NULL,
    avatar_kind character varying DEFAULT 'default'::character varying NOT NULL,
    gravatar_email character varying DEFAULT ''::character varying NOT NULL,
    gravatar_url character varying DEFAULT ''::character varying NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL,
    discarded_at timestamp(6) without time zone,
    last_post_at timestamp without time zone
);
```

#### actors テーブル（既存）

```sql
CREATE TABLE public.actors (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    profile_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);
```

#### user_profiles テーブル（既存）

```sql
CREATE TABLE public.user_profiles (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    user_id uuid NOT NULL,
    profile_id uuid NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);
```

#### email_confirmations テーブル（既存）

```sql
CREATE TABLE public.email_confirmations (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    email public.citext NOT NULL,
    event character varying NOT NULL,
    code character varying NOT NULL,
    succeeded_at timestamp(6) without time zone,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
);
```

#### rate_limits テーブル（新規）

```sql
CREATE TABLE rate_limits (
    id VARCHAR NOT NULL PRIMARY KEY DEFAULT generate_ulid(),
    key VARCHAR NOT NULL,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(key, window_start)
);

-- インデックス
CREATE INDEX idx_rate_limits_key_window_start ON rate_limits(key, window_start);
CREATE INDEX idx_rate_limits_window_start ON rate_limits(window_start);
```

### API 設計（ルーティング）

| URL                   | メソッド | ハンドラー        | 説明                           |
| --------------------- | -------- | ----------------- | ------------------------------ |
| `/sign_up`            | GET      | `sign_up.New`     | メールアドレス入力フォーム表示 |
| `/sign_up`            | POST     | `sign_up.Create`  | 確認コード送信処理             |
| `/email_confirmation` | GET      | （既存）          | 確認コード入力フォーム表示     |
| `/email_confirmation` | POST     | （既存を拡張）    | 確認コード検証処理             |
| `/accounts/new`       | GET      | `accounts.New`    | アカウント詳細入力フォーム表示 |
| `/accounts`           | POST     | `accounts.Create` | アカウント作成処理             |

### コード設計

#### ディレクトリ構造

```
internal/
├── handler/
│   ├── sign_up/
│   │   ├── handler.go      # Handler構造体と依存性
│   │   ├── new.go          # GET /sign_up
│   │   ├── create.go       # POST /sign_up
│   │   └── validator.go    # CreateValidator バリデーション
│   └── accounts/
│       ├── handler.go      # Handler構造体と依存性
│       ├── new.go          # GET /accounts/new
│       ├── create.go       # POST /accounts
│       └── validator.go    # CreateValidator バリデーション
├── usecase/
│   └── create_account.go   # アカウント作成ロジック
├── repository/
│   ├── profile_repository.go   # プロフィール CRUD
│   └── user_profile_repository.go  # ユーザープロフィール関連付け
├── model/
│   ├── profile.go          # プロフィールモデル
│   └── user_profile.go     # ユーザープロフィールモデル
├── templates/
│   └── pages/
│       ├── sign_up/
│       │   └── new.templ   # メールアドレス入力フォーム
│       └── accounts/
│           └── new.templ   # アカウント詳細入力フォーム
└── i18n/
    └── locales/
        ├── ja.toml         # 日本語翻訳（追加）
        └── en.toml         # 英語翻訳（追加）
```

#### 主要な構造体

**sign_up/handler.go**

```go
type Handler struct {
    cfg                       *config.Config
    sessionMgr                *session.Manager
    userRepo                  *repository.UserRepository
    createEmailConfirmationUC *usecase.CreateEmailConfirmationUsecase
    turnstileClient           *turnstile.Client
}
```

**sign_up/validator.go**

```go
type CreateValidator struct {
    userRepo *repository.UserRepository
}

type CreateValidatorInput struct {
    Email string
}

type CreateValidatorResult struct {
    FormErrors *session.FormErrors
    Err        error
}
```

**accounts/handler.go**

```go
type Handler struct {
    cfg                      *config.Config
    sessionMgr               *session.Manager
    emailConfirmationRepo    *repository.EmailConfirmationRepository
    createAccountUC          *usecase.CreateAccountUsecase
    createSessionUC          *usecase.CreateSessionUsecase
    turnstileClient          *turnstile.Client
}
```

**accounts/validator.go**

```go
type CreateValidator struct {
    userRepo    *repository.UserRepository
    profileRepo *repository.ProfileRepository
}

type CreateValidatorInput struct {
    Atname   string
    Password string
}

type CreateValidatorResult struct {
    FormErrors *session.FormErrors
    Err        error
}
```

**usecase/create_account.go**

```go
type CreateAccountUsecase struct {
    db              *sql.DB
    userRepo        *repository.UserRepository
    profileRepo     *repository.ProfileRepository
    userProfileRepo *repository.UserProfileRepository
    actorRepo       *repository.ActorRepository
}

type CreateAccountInput struct {
    Email    string
    Atname   string
    Password string
    Locale   string
    TimeZone string
}

type CreateAccountOutput struct {
    Actor *model.Actor
}

func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error)
```

### サインアップフロー

```
1. ユーザーが GET /sign_up にアクセス
   ├── 認証済みの場合 → / にリダイレクト
   └── 未認証の場合 → メールアドレス入力フォーム表示

2. ユーザーがメールアドレスを送信 (POST /sign_up)
   ├── CSRF トークン検証
   ├── Turnstile 検証
   ├── フォームバリデーション
   │   ├── メールアドレス: 必須、形式チェック
   │   └── メールアドレス: 重複チェック（既に登録済みならエラー）
   ├── 確認コード生成・送信 (CreateEmailConfirmationUsecase)
   │   ├── 6桁の確認コード生成
   │   ├── email_confirmations テーブルに INSERT
   │   └── メール送信 (Resend API)
   ├── セッションに email_confirmation_id を保存
   ├── フラッシュメッセージ設定
   └── /email_confirmation にリダイレクト

3. ユーザーが確認コードを入力 (既存の email_confirmation ハンドラー)
   ├── 確認コード検証
   ├── 成功時: email_confirmations.succeeded_at を更新
   ├── event が "sign_up" の場合 → /accounts/new にリダイレクト
   └── その他の event → 対応するページにリダイレクト

4. ユーザーが GET /accounts/new にアクセス
   ├── セッションから email_confirmation_id を取得
   ├── email_confirmation が存在しない or 未確認 → / にリダイレクト
   └── アカウント詳細入力フォーム表示（メールアドレスは読み取り専用）

5. ユーザーがアカウント詳細を送信 (POST /accounts)
   ├── CSRF トークン検証
   ├── Turnstile 検証
   ├── フォームバリデーション
   │   ├── アットネーム: 必須、フォーマット、長さ、重複、予約名チェック
   │   └── パスワード: 必須、最小長チェック
   ├── アカウント作成 (CreateAccountUsecase)
   │   ├── トランザクション開始
   │   ├── ProfileRecord 作成
   │   ├── UserRecord 作成（パスワードは bcrypt ハッシュ化）
   │   ├── UserProfileRecord 作成
   │   ├── ActorRecord 作成
   │   └── トランザクションコミット
   ├── セッション作成 (CreateSessionUsecase)
   ├── Cookie にセッショントークンを設定
   ├── セッションから email_confirmation_id を削除
   ├── フラッシュメッセージ設定
   └── / にリダイレクト
```

### セキュリティ設計

#### パスワードハッシュ化

Rails の `has_secure_password` と同じく bcrypt を使用：

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}
```

#### バリデーションルール

**メールアドレス**:

- 必須
- メール形式チェック（`net/mail.ParseAddress`）
- 重複チェック（users テーブル検索）

**アットネーム**:

- 必須
- フォーマット: `/^[A-Za-z0-9_]+$/`（半角英数字とアンダースコアのみ）
- 長さ: 最大20文字
- 重複チェック（profiles テーブル検索）
- 予約名チェック（admin, support, help など）

**パスワード**:

- 必須
- 最小長: 8文字
- 最大長: 72文字（bcrypt の制限）

#### 予約アットネーム

以下のアットネームは予約済みとして登録不可とする：

```go
var ReservedAtnames = []string{
    "admin", "administrator", "support", "help", "info",
    "contact", "sales", "marketing", "noreply", "no-reply",
    "postmaster", "webmaster", "root", "system", "api",
    "mewst", "official", "news", "blog", "status",
}
```

### テスト戦略

- **ハンドラーテスト**: HTTP リクエスト・レスポンスの統合テスト
- **ユースケーステスト**: アカウント作成ロジックのテスト
- **リポジトリテスト**: DB 操作のテスト
- **バリデーションテスト**: 各バリデーションルールのテスト

テストでは実際の PostgreSQL データベースを使用し、トランザクションで分離します。

## タスクリスト

### フェーズ 1: 基盤整備

- [x] **1-1**: [Go] リポジトリ層の実装（Profile, UserProfile）
  - `internal/repository/profile_repository.go` の作成
  - `internal/repository/user_profile_repository.go` の作成
  - `internal/model/profile.go` の作成
  - `internal/model/user_profile.go` の作成
  - sqlc クエリの追加（`db/queries/profiles.sql`, `db/queries/user_profiles.sql`）
  - **想定ファイル数**: 約 8 ファイル（実装 6 + テスト 2）
  - **想定行数**: 約 400 行（実装 250 行 + テスト 150 行）

- [x] **1-2**: [Go] アカウント作成ユースケースの実装
  - `internal/usecase/create_account.go` の作成
  - トランザクション管理
  - Profile, User, UserProfile, Actor の一括作成
  - bcrypt パスワードハッシュ化
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 300 行（実装 120 行 + テスト 180 行）
  - **備考**: 重複データ（メールアドレス・アットネーム）のバリデーションは Handler 層（4-1 タスク）で実装するため、Usecase 層では異常系テストを省略。4-1 タスクで必ず以下の異常系テストを実装すること：
    - 既に登録済みのメールアドレスでの登録エラー
    - 既に使用されているアットネームでの登録エラー

- [x] **1-3**: [Go] レート制限の実装（Wikino の実装を移植）
  - `db/migrations/YYYYMMDDHHMMSS_create_rate_limits.sql` の作成
  - `db/queries/rate_limits.sql` の作成
  - `internal/ratelimit/limiter.go` の作成
  - sqlc コード生成
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 2: サインアップフォーム（メールアドレス入力）

- [x] **2-1**: [Go] サインアップハンドラーの実装
  - `internal/handler/sign_up/handler.go` の作成
  - `internal/handler/sign_up/new.go` の作成
  - `internal/handler/sign_up/create.go` の作成
  - `internal/handler/sign_up/validator.go` の作成
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 500 行（実装 200 行 + テスト 300 行）

- [x] **2-2**: [Go] サインアップテンプレートの実装
  - `internal/templates/pages/sign_up/new.templ` の作成
  - 国際化対応（`ja.toml`, `en.toml` への追加）
  - **想定ファイル数**: 約 3 ファイル（実装 3 + テスト 0）
  - **想定行数**: 約 150 行（実装 150 行 + テスト 0 行）

### フェーズ 3: メール確認フローの拡張

- [x] **3-1**: [Go] email_confirmation ハンドラーの拡張
  - event が "sign_up" の場合のリダイレクト先を `/accounts/new` に変更
  - セッションへの email_confirmation_id 保存処理の追加
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 100 行（実装 40 行 + テスト 60 行）

### フェーズ 4: アカウント作成フォーム

- [x] **4-1**: [Go] アカウントハンドラーの実装
  - `internal/handler/accounts/handler.go` の作成
  - `internal/handler/accounts/new.go` の作成
  - `internal/handler/accounts/create.go` の作成
  - `internal/handler/accounts/validator.go` の作成
  - 予約アットネームチェックの実装
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 600 行（実装 250 行 + テスト 350 行）

- [x] **4-2**: [Go] アカウントテンプレートの実装
  - `internal/templates/pages/accounts/new.templ` の作成
  - 国際化対応（`ja.toml`, `en.toml` への追加）
  - **想定ファイル数**: 約 3 ファイル（実装 3 + テスト 0）
  - **想定行数**: 約 200 行（実装 200 行 + テスト 0 行）

### フェーズ 5: 統合とルーティング

- [x] **5-1**: [Go] ルーティング設定とミドルウェア更新
  - ルーティング設定（`/sign_up`, `/accounts/new`, `/accounts`）
  - リバースプロキシのホワイトリスト更新
  - RequireNoAuth ミドルウェアの適用
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 150 行（実装 80 行 + テスト 70 行）

### フェーズ 6: Rails 版のセッションクッキー署名削除（オプション）

<!--
この作業は、ログイン機能のときに完了している可能性があります。
`@docs/designs/3_done/202601/go.md` の「Rails版: セッションクッキーの署名を削除」セクションを参照してください。
-->

- [x] **6-1**: [Rails] セッションクッキーの署名削除
  - `cookies.signed` を `cookies` に変更
  - 既存ユーザーへの影響確認（再ログインが必要になる）
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 50 行（実装 10 行 + テスト 40 行）

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **サインアップ停止機能（SignUpStopper）**: Rails 版には存在するが、Go 版では当面不要
- **ソーシャルログイン（OAuth）**: 別タスクで実装予定
- **メール確認の再送機能**: 別タスクで実装予定
- **プロフィール画像設定**: 別タスクで実装予定

## 参考資料

- **Rails 版 Mewst サインアップ実装**: `/workspace/rails/app/controllers/sign_up/`, `/workspace/rails/app/controllers/accounts/`
- **Go 版 Mewst ログイン実装**: `/workspace/go/internal/handler/sign_in/`
- **Go 版 Mewst メール確認実装**: `/workspace/go/internal/handler/email_confirmation/`
- **Annict Go 版サインアップ実装**: `/annict/go/internal/handler/sign_up/`
- **Wikino レート制限実装**: `/wikino/go/internal/ratelimit/limiter.go`
- **Go CLAUDE.md**: `/workspace/go/CLAUDE.md`
- [chi ルーター](https://github.com/go-chi/chi)
- [sqlc](https://docs.sqlc.dev/)
- [templ](https://templ.guide/)
- [Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/)
- [bcrypt (golang.org/x/crypto)](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
