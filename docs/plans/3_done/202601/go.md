# Go への移行 (ログイン機能編) 仕様書

<!--
このテンプレートの使い方:
1. このファイルを `.claude/specs/2_todo/` ディレクトリにコピー
   例: cp .claude/specs/template.md .claude/specs/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残しておくことを推奨
-->

## 概要

<!--
ガイドライン:
- この機能が「何を」実現するのかを簡潔に説明
- ユーザーにとっての価値や背景を記述
- 2-3段落程度で簡潔に
-->

Go 版 Mewst にログイン機能を実装します。ユーザーはメールアドレスとパスワードを入力してログインし、セッションが作成されることで認証済み状態となります。

Rails 版 Mewst と同じ DB・セッションストアを共有するため、Rails 版で作成されたユーザーアカウントでログインでき、Go 版でログインしたセッションは Rails 版でも有効です。

**目的**:

- Rails 版から Go 版への段階的移行の第一歩として、認証基盤を整備する
- ユーザーが Go 版でログインできるようにする
- Rails 版と Go 版でセッションを共有し、シームレスな移行を実現する

**背景**:

- Mewst は Rails 版から Go 版への移行プロジェクトを進めている
- ログイン機能は他の機能の前提となる認証基盤であり、最初に実装する必要がある
- Rails 版の既存セッション管理（`sessions` テーブル + `has_secure_token`）を Go 版でも使用する

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

<!--
「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
箇条書きで簡潔に
-->

- ユーザーはメールアドレスとパスワードを入力してログインできる
- ログイン済みユーザーがログインページにアクセスした場合、ホームページにリダイレクトする
- ログインに成功すると、セッションが作成され、Cookie にセッショントークンが設定される
- ログインに失敗した場合（メールアドレスまたはパスワードが間違っている場合）、エラーメッセージを表示する
- ユーザーはログアウトできる
- ログアウトするとセッションクッキーが削除される
- ログイン・ログアウト後にフラッシュメッセージを表示する

### 非機能要件

<!--
必要に応じて以下のような項目を追加してください：
- セキュリティ（認証、認可、暗号化、監査ログなど）
- パフォーマンス（応答時間、スループット、リソース使用量など）
- ユーザビリティ（UX）（使いやすさ、わかりやすさ、アクセシビリティなど）
- 可用性・信頼性（稼働率、障害時の挙動、エラーハンドリングなど）
- 保守性（テストのしやすさ、コードの読みやすさ、ドキュメントなど）

不要な場合はこのセクション全体を削除してください。
-->

#### セキュリティ

- パスワードは bcrypt でハッシュ化されて保存されている（Rails 版の既存仕様）
- CSRF 対策を実施する（全フォームに CSRF トークンを含める）
- セッショントークンは `has_secure_token` で生成される安全なトークンを使用
- Cookie には `httponly=true`, `same_site=lax` を設定
- Cloudflare Turnstile による Bot 対策を実施
- ログイン試行時に IP アドレスと User-Agent を記録

#### 国際化

- 日本語と英語の両言語に対応
- エラーメッセージ、フラッシュメッセージ、フォームラベルを国際化

#### Rails 互換性

- Rails 版と同じ `sessions` テーブルを使用
- Rails 版と同じ Cookie キー（`mewst_session_token`）を使用
- Rails 版で作成されたセッションは Go 版でも有効

## 設計

<!--
ガイドライン:
- 技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など）
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど）
  - テスト戦略（単体テスト、統合テスト、E2Eテストの方針）
  - マイグレーション管理（データベースマイグレーションの方針）
  - 実装方針（特記事項、既存システムとの関係、制約など）

不要な場合はこのセクション全体を削除してください。
-->

### 技術スタック

- **パスワード検証**: `golang.org/x/crypto/bcrypt`
- **セッショントークン生成**: `crypto/rand` + `encoding/hex`
- **HTTP ルーター**: `chi/v5`
- **テンプレート**: `templ`
- **DB アクセス**: `sqlc`
- **Bot 対策**: Cloudflare Turnstile

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

#### sessions テーブル（既存）

```sql
CREATE TABLE public.sessions (
    id uuid DEFAULT public.generate_ulid() NOT NULL,
    actor_id uuid NOT NULL,
    token character varying NOT NULL,
    ip_address character varying DEFAULT ''::character varying NOT NULL,
    user_agent character varying DEFAULT ''::character varying NOT NULL,
    signed_in_at timestamp(6) without time zone NOT NULL,
    created_at timestamp(6) without time zone NOT NULL,
    updated_at timestamp(6) without time zone NOT NULL
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

### API 設計（ルーティング）

| URL         | メソッド | ハンドラー        | 説明                 |
| ----------- | -------- | ----------------- | -------------------- |
| `/sign_in`  | GET      | `sign_in.New`     | ログインフォーム表示 |
| `/sign_in`  | POST     | `sign_in.Create`  | ログイン処理         |
| `/sign_out` | DELETE   | `sign_out.Delete` | ログアウト処理       |

### コード設計

#### ディレクトリ構造

```
internal/
├── handler/
│   ├── sign_in/
│   │   ├── handler.go      # Handler構造体と依存性
│   │   ├── new.go          # GET /sign_in
│   │   ├── create.go       # POST /sign_in
│   │   └── request.go      # CreateRequest バリデーション
│   └── sign_out/
│       ├── handler.go      # Handler構造体と依存性
│       └── delete.go       # DELETE /sign_out
├── usecase/
│   └── create_session.go   # セッション作成ロジック
├── repository/
│   ├── user_repository.go      # ユーザー取得
│   ├── session_repository.go   # セッション CRUD
│   └── actor_repository.go     # アクター取得
├── session/
│   ├── manager.go          # セッション管理
│   └── flash.go            # フラッシュメッセージ
├── auth/
│   └── password.go         # bcrypt パスワード検証
├── middleware/
│   ├── auth.go             # 認証ミドルウェア
│   └── csrf.go             # CSRF ミドルウェア
├── turnstile/
│   └── client.go           # Turnstile クライアント
└── templates/
    └── pages/
        └── sign_in/
            └── new.templ   # ログインフォーム
```

#### 主要な構造体

**Handler（sign_in/handler.go）**

```go
type Handler struct {
    cfg              *config.Config
    sessionMgr       *session.Manager
    userRepo         *repository.UserRepository
    actorRepo        *repository.ActorRepository
    createSessionUC  *usecase.CreateSessionUsecase
    turnstileClient  *turnstile.Client
}
```

**CreateRequest（sign_in/request.go）**

```go
type CreateRequest struct {
    Email    string
    Password string
}

func (r *CreateRequest) Validate(ctx context.Context) *session.FormErrors {
    // メールアドレス必須・形式チェック
    // パスワード必須チェック
}
```

**CreateSessionUsecase（usecase/create_session.go）**

```go
type CreateSessionUsecase struct {
    sessionRepo *repository.SessionRepository
}

type CreateSessionInput struct {
    ActorID   string
    IPAddress string
    UserAgent string
}

type CreateSessionOutput struct {
    Token string
}

func (uc *CreateSessionUsecase) Execute(ctx context.Context, input CreateSessionInput) (*CreateSessionOutput, error)
```

**Manager（session/manager.go）**

```go
type Manager struct {
    sessionRepo *repository.SessionRepository
    cfg         *config.Config
}

func (m *Manager) GetCurrentUser(ctx context.Context, r *http.Request) (*model.User, error)
func (m *Manager) SetSessionCookie(w http.ResponseWriter, r *http.Request, token string)
func (m *Manager) DeleteSessionCookie(w http.ResponseWriter, r *http.Request)
```

### 認証フロー

```
1. ユーザーが GET /sign_in にアクセス
   ├── 認証済みの場合 → / にリダイレクト
   └── 未認証の場合 → ログインフォーム表示

2. ユーザーがフォーム送信 (POST /sign_in)
   ├── CSRF トークン検証
   ├── Turnstile 検証
   ├── フォームバリデーション
   │   ├── メールアドレス: 必須、形式チェック
   │   └── パスワード: 必須
   ├── ユーザー検索 (email で検索)
   │   └── 見つからない場合 → エラー表示
   ├── パスワード検証 (bcrypt)
   │   └── 不一致の場合 → エラー表示
   ├── アクター取得
   ├── セッション作成 (CreateSessionUsecase)
   │   ├── セッショントークン生成 (has_secure_token 互換)
   │   ├── sessions テーブルに INSERT
   │   └── IP/User-Agent 記録
   ├── Cookie にセッショントークンを設定
   ├── フラッシュメッセージ設定
   └── / にリダイレクト

3. ユーザーが DELETE /sign_out にアクセス
   ├── セッションクッキーを削除
   ├── フラッシュメッセージ設定
   └── / にリダイレクト
```

### セキュリティ設計

#### セッショントークン生成

Rails の `has_secure_token` と互換性のある形式でトークンを生成：

```go
func GenerateSecureToken() (string, error) {
    // 24バイトのランダムデータを生成し、Base64 URL-safe エンコード
    b := make([]byte, 24)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}
```

#### Cookie 設定

```go
cookie := &http.Cookie{
    Name:     "mewst_session_token",
    Value:    token,
    Path:     "/",
    Domain:   cfg.CookieDomain,
    Secure:   true,
    HttpOnly: true,
    SameSite: http.SameSiteLaxMode,
    MaxAge:   10 * 365 * 24 * 60 * 60, // 10年（Rails版と同じ）
}
```

### テスト戦略

- **ハンドラーテスト**: HTTP リクエスト・レスポンスの統合テスト
- **ユースケーステスト**: セッション作成ロジックのテスト
- **リポジトリテスト**: DB 操作のテスト
- **認証テスト**: パスワード検証、セッション管理のテスト

テストでは実際の PostgreSQL データベースを使用し、トランザクションで分離します。

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- チェックボックスで進捗を管理
- **重要**: 1タスク = 1 Pull Request の粒度で作成してください
- **重要**: 各タスクには想定ファイル数と想定行数を明記してください（PRサイズの見積もりのため）
- 想定ファイル数は「実装」と「テスト」に分けて記載してください
- 想定行数も「実装」と「テスト」に分けて記載してください
- 依存関係を明確に
- Pull Requestのガイドラインは CLAUDE.md を参照（変更ファイル数20以下、変更行数300行以下）

タスク番号の付け方:
- 各タスクには階層的な番号を付与します（例: 1-1, 1-2, 2-1, 2-2）
- フォーマット: **フェーズ番号-タスク番号**: タスク名
- タスクの前に別のタスクを追加する場合は、サブ番号を使用します
  - 例: タスク 2-1 の前にタスクを追加する場合 → 2-0
  - 例: タスク 2-0 の前にタスクを追加する場合 → 2-0-1
- この番号はブランチ名の一部として使用されます（例: feature-1-1, feature-2-0）
-->

### フェーズ 1: プロジェクト基盤整備

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
-->

- [x] **1-0**: 開発環境コマンド（Makefile）の作成
  - `Makefile` の作成（run, build, test, fmt, lint など）
  - `.golangci.yml` の作成（golangci-lint 設定）
  - `tools.go` の作成（開発ツールの依存関係管理）
  - `.air.toml` の作成（ホットリロード設定）
  - **想定ファイル数**: 約 4 ファイル（実装 4 + テスト 0）
  - **想定行数**: 約 250 行（実装 250 行 + テスト 0 行）

- [x] **1-0-1**: GitHub Actions CI の設定
  - `.github/workflows/go-ci.yml` の作成（Go 版 CI）
    - Lint ジョブ: templ generate チェック、go mod tidy チェック、golangci-lint
    - Test ジョブ: PostgreSQL サービス、テスト実行
    - Build ジョブ: バイナリビルド確認
  - `.github/workflows/rails-ci.yml` の更新（Rails 版 CI）
    - working-directory を `rails/` に設定
    - paths トリガーを追加（`rails/**`, `.github/workflows/rails-ci.yml`）
  - **参考**: Annict の設定（`/annict/.github/workflows/go-ci.yml`, `/annict/.github/workflows/rails-ci.yml`）
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 300 行（実装 300 行 + テスト 0 行）

- [x] **1-1**: Go プロジェクトの初期化
  - `go.mod`, `go.sum` の作成
  - 基本的なディレクトリ構造の作成
  - 必要なライブラリのインストール（chi, sqlc, templ, bcrypt など）
  - `cmd/server/main.go` エントリポイント作成
  - **想定ファイル数**: 約 8 ファイル（実装 8 + テスト 0）
  - **想定行数**: 約 250 行（実装 250 行 + テスト 0 行）

- [x] **1-2**: 設定管理の実装
  - `internal/config/config.go` の作成
  - 環境変数からの設定読み込み
  - `.env.example` の作成
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

- [x] **1-3**: データベース接続とマイグレーション
  - DB 接続設定
  - `db/schema.sql` の作成（Rails 版から移植）
  - `sqlc.yaml` の設定
  - 基本クエリの定義（users, sessions, actors）
  - **想定ファイル数**: 約 5 ファイル（実装 5 + テスト 0）
  - **想定行数**: 約 300 行（実装 300 行 + テスト 0 行）

### フェーズ 2: 認証基盤の実装

- [x] **2-1**: リポジトリ層の実装
  - `internal/repository/user_repository.go`
  - `internal/repository/session_repository.go`
  - `internal/repository/actor_repository.go`
  - **想定ファイル数**: 約 6 ファイル（実装 3 + テスト 3）
  - **想定行数**: 約 400 行（実装 150 行 + テスト 250 行）

- [x] **2-2**: パスワード検証の実装
  - `internal/auth/password.go`
  - bcrypt によるパスワード検証
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 100 行（実装 30 行 + テスト 70 行）

- [x] **2-3**: セッション管理の実装
  - `internal/session/manager.go`
  - `internal/session/flash.go`
  - セッショントークン生成
  - Cookie 設定・取得
  - 現在のユーザー取得
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **2-4**: 認証ミドルウェアの実装
  - `internal/middleware/auth.go`
  - `RequireAuth` ミドルウェア
  - `RequireNoAuth` ミドルウェア
  - コンテキストへのユーザー情報設定
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 200 行（実装 80 行 + テスト 120 行）

### フェーズ 3: ログイン機能の実装

- [x] **3-1**: CSRF ミドルウェアの実装
  - `internal/middleware/csrf.go`
  - CSRF トークン生成・検証
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 150 行（実装 60 行 + テスト 90 行）

- [x] **3-2**: Turnstile クライアントの実装
  - `internal/turnstile/client.go`
  - Cloudflare Turnstile API との連携
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 150 行（実装 60 行 + テスト 90 行）

- [x] **3-3**: セッション作成ユースケースの実装
  - `internal/usecase/create_session.go`
  - セッショントークン生成
  - sessions テーブルへの INSERT
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 200 行（実装 80 行 + テスト 120 行）

- [x] **3-4**: ログインフォームテンプレートの実装
  - `internal/templates/pages/sign_in/new.templ`
  - `internal/templates/layouts/simple.templ`
  - `internal/templates/components/` 共通コンポーネント
  - `.github/workflows/go-ci.yml` の `Check templ generate` のコメントアウトを解除
  - **想定ファイル数**: 約 5 ファイル（実装 5 + テスト 0）
  - **想定行数**: 約 250 行（実装 250 行 + テスト 0 行）

- [x] **3-5**: ログインハンドラーの実装
  - `internal/handler/sign_in/handler.go`
  - `internal/handler/sign_in/new.go`
  - `internal/handler/sign_in/create.go`
  - `internal/handler/sign_in/request.go`
  - **想定ファイル数**: 約 8 ファイル（実装 4 + テスト 4）
  - **想定行数**: 約 600 行（実装 250 行 + テスト 350 行）

### フェーズ 4: ログアウト機能の実装

- [x] **4-1**: ログアウトハンドラーの実装
  - `internal/handler/sign_out/handler.go`
  - `internal/handler/sign_out/delete.go`
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 200 行（実装 60 行 + テスト 140 行）

### フェーズ 5: 統合とルーティング

- [x] **5-1**: ルーティング設定とリバースプロキシミドルウェアの更新
  - ルーティング設定（`/sign_in`, `/sign_out`）
  - リバースプロキシのホワイトリスト更新
  - Method Override ミドルウェア
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

- [x] **5-2**: 国際化（I18n）の実装
  - `internal/i18n/i18n.go`
  - `internal/i18n/locales/ja.toml`
  - `internal/i18n/locales/en.toml`
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 200 行（実装 150 行 + テスト 50 行）

### フェーズ 6: フロントエンド基盤の整備

- [x] **6-1**: JS/CSS ビルド環境の構築
  - `package.json` の作成（pnpm 設定、scripts、devDependencies）
    - `@tailwindcss/cli`: Tailwind CSS v4 CLI
    - `tailwindcss`: Tailwind CSS v4
    - `basecoat-css`: UI コンポーネントライブラリ
    - `esbuild`: JavaScript バンドラー
    - `concurrently`: watch タスクの並行実行
  - `tsconfig.json` の作成（TypeScript 設定）
  - `web/style.css` の作成（Tailwind CSS v4 ソースファイル）
  - `web/main.js` の作成（JavaScript エントリーポイント）
  - `static/css/`, `static/js/` ディレクトリの作成
  - `.gitignore` の更新（ビルド成果物、node_modules）
  - **参考**: Annict Go 版の設定（`/annict/go/package.json`）
  - **想定ファイル数**: 約 6 ファイル（実装 6 + テスト 0）
  - **想定行数**: 約 150 行（実装 150 行 + テスト 0 行）

- [x] **6-2**: head 要素内の更新
- [x] **6-3**: `/manifest.json` の実装
- [x] **6-4**: ログインページの修正

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **新規ユーザー登録（サインアップ）**: 別タスクで実装予定
- **パスワードリセット**: 別タスクで実装予定
- **ソーシャルログイン（OAuth）**: 別タスクで実装予定
- **メール認証（6 桁コードログイン）**: Mewst はパスワードログインのみ
- **Remember Me 機能**: 現在の Cookie 有効期限（10 年）で十分
- **セッションの明示的な削除（DB から DELETE）**: Rails 版も削除していない
- **レート制限**: 将来的に実装予定

## Rails版: セッションクッキーの署名を削除

### 概要

Go版とRails版でセッションを共有するため、Rails側の`cookies.signed`を`cookies`に変更する。

これにより、Go側で`SECRET_KEY_BASE`の共有が不要になり、実装がシンプルになる。

### 背景

現在のRails実装では`cookies.signed`を使用しており、クッキー値がHMAC-SHA256で署名されている。
Go側でこの署名を検証するには`SECRET_KEY_BASE`を共有し、同じ鍵導出処理を実装する必要がある。

Annictでは`ActiveRecord::SessionStore`を使用しており、クッキーには署名なしのセッションIDが格納されている。
Mewstも同様のアプローチに変更することで、Go側の実装を簡素化できる。

### セキュリティについて

- セッショントークンは`has_secure_token`で生成されるランダムな文字列（24文字）
- 攻撃者がトークンを偽造しても、DBに存在しないトークンは無効
- Annictで実績のあるアプローチ

### 変更対象ファイル

`app/controllers/controller_concerns/authenticatable.rb`

### 変更内容

#### 1. `sign_in`メソッド（14行目付近）

```ruby
# 変更前
cookies.signed.permanent[SessionRecord::COOKIE_KEY] = {
  value: session.token,
  httponly: true,
  same_site: :lax
}

# 変更後
cookies.permanent[SessionRecord::COOKIE_KEY] = {
  value: session.token,
  httponly: true,
  same_site: :lax
}
```

#### 2. `viewer`メソッド（31-33行目付近）

```ruby
# 変更前
@viewer ||= T.let(begin
  return unless cookies.signed[SessionRecord::COOKIE_KEY]
  SessionRecord.find_by(token: cookies.signed[SessionRecord::COOKIE_KEY])&.actor_record
end, T.nilable(ActorRecord))

# 変更後
@viewer ||= T.let(begin
  return unless cookies[SessionRecord::COOKIE_KEY]
  SessionRecord.find_by(token: cookies[SessionRecord::COOKIE_KEY])&.actor_record
end, T.nilable(ActorRecord))
```

### 影響

- 既存ユーザーは一度ログアウトされる（署名付きクッキーが読めなくなるため）
- 再ログインで新しいセッションが作成される

### Go側の変更（この変更後に実施）

Rails側の変更が完了したら、Go側で以下の変更を行う：

1. `internal/session/rails_cookie.go` を削除
2. `internal/session/rails_cookie_test.go` を削除
3. `internal/session/manager.go` から署名処理を削除
4. `internal/config/config.go` から `SecretKeyBase` を削除
5. 各テストファイルから `SecretKeyBase` を削除

### 確認手順

1. Rails版でログイン → クッキーが設定される
2. Go版のページにアクセス → セッションが共有されていることを確認
3. Go版でログアウト → Rails版もログアウトされていることを確認

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- **Rails 版 Mewst ログイン実装**: `/workspace/rails/app/controllers/sessions/`
- **Annict Go 版ログイン実装**: `/annict/go/internal/handler/sign_in/`
- **Go CLAUDE.md**: `/workspace/go/CLAUDE.md`
- [chi ルーター](https://github.com/go-chi/chi)
- [sqlc](https://docs.sqlc.dev/)
- [templ](https://templ.guide/)
- [Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/)
