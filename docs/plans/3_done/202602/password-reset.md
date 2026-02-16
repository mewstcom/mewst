# パスワードリセット機能 仕様書

<!--
このテンプレートの使い方:
1. このファイルを `docs/specs/2_todo/` ディレクトリにコピー
   例: cp docs/specs/template.md docs/specs/2_todo/new-feature.md
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

パスワードを忘れたユーザーが、メールアドレスを使用してパスワードをリセットできる機能です。ユーザーは登録済みのメールアドレスを入力し、送信された確認コードを入力することで、新しいパスワードを設定できます。

この機能は既にRails版で実装されており、Go版への移行として再実装します。Rails版と同じデータベーステーブル（`email_confirmations`）を共有し、同じフローを維持します。

**目的**:

- パスワードを忘れたユーザーがアカウントにアクセスできるようにする
- セキュアな方法でパスワードリセットを実現する（メール確認による本人確認）
- Rails版からGo版への段階的移行の一環として実装

**背景**:

- Rails版で実装済みの機能をGo版に移行するプロジェクトの一環
- ログイン機能はGo版で実装済みのため、パスワードリセット機能も移行が必要
- Rails版と同じデータベーステーブル・セッションストアを共有することで、移行期間中も両システムが共存可能

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

- ユーザーは `/password_reset` ページでメールアドレスを入力し、パスワードリセットを開始できる
- システムは入力されたメールアドレス宛に6桁の確認コードを含むメールを送信する
- ユーザーは `/email_confirmation` ページで受信した確認コードを入力して本人確認を行う
- ユーザーは本人確認完了後、`/password/edit` ページで新しいパスワードを設定できる
- システムはパスワード更新後、既存セッションを無効化しログインページにリダイレクトする
- ログイン済みユーザーがパスワードリセットページにアクセスした場合、ホームページにリダイレクトする
- 確認コードは15分で有効期限切れとなる
- 確認コードが一致しない、または有効期限切れの場合はエラーを表示する

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

- **CSRF対策**: すべてのフォーム送信でCSRFトークンを検証する
- **Bot対策**: パスワードリセット開始フォームにCloudflare Turnstileを導入する
- **確認コード**: 6桁のランダムな数字を使用（Rails版と同じ形式）
- **有効期限**: 確認コードは作成から15分で失効する
- **パスワードハッシュ化**: bcryptを使用してパスワードをハッシュ化する（既存実装を使用）
- **セッション無効化**: パスワード更新後は既存のセッションをすべて無効化する
- **メールアドレス漏洩対策**: 存在しないメールアドレスでも「メールを送信しました」と表示し、メールアドレスの存在確認を防ぐ

#### ユーザビリティ

- 日本語と英語の両方に対応（i18n）
- フォームバリデーションエラーは明確なメッセージで表示
- 確認コード入力は数字入力フィールドを使用
- パスワード要件（最小8文字）をフォームに表示

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

### ユーザーフロー

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. パスワードリセット開始                                         │
│    GET /password_reset → メールアドレス入力フォーム表示           │
│    POST /password_reset → 確認コード生成 + メール送信              │
│                        → セッションにemail_confirmation_id保存    │
│                        → /email_confirmation へリダイレクト        │
└─────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────┐
│ 2. 確認コード入力                                                │
│    GET /email_confirmation → 確認コード入力フォーム表示           │
│    POST /email_confirmation → 確認コード検証                      │
│                            → 成功: succeeded_at を記録            │
│                            → /password/edit へリダイレクト         │
│                            → 失敗: エラーメッセージ表示            │
└─────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────┐
│ 3. 新しいパスワード設定                                          │
│    GET /password/edit → 新しいパスワード入力フォーム表示          │
│    PATCH /password → パスワード更新                               │
│                    → セッションリセット                           │
│                    → /sign_in へリダイレクト                       │
└─────────────────────────────────────────────────────────────────┘
```

### ルーティング設計

| メソッド | パス | ハンドラー | 説明 |
|---------|------|-----------|------|
| GET | `/password_reset` | `password_reset.New` | メールアドレス入力フォーム表示 |
| POST | `/password_reset` | `password_reset.Create` | 確認コード生成・メール送信 |
| GET | `/email_confirmation` | `email_confirmation.New` | 確認コード入力フォーム表示 |
| POST | `/email_confirmation` | `email_confirmation.Create` | 確認コード検証 |
| GET | `/password/edit` | `password.Edit` | 新しいパスワード入力フォーム表示 |
| PATCH | `/password` | `password.Update` | パスワード更新 |

### データベース設計

Rails版の既存テーブル `email_confirmations` を共有して使用します。

**email_confirmations テーブル**（既存）

| カラム | 型 | 説明 |
|--------|------|------|
| id | uuid (ULID) | 主キー |
| email | citext | メールアドレス（大文字小文字区別なし） |
| event | varchar | イベント種別（`password_reset`, `sign_up`, `email_update`） |
| code | varchar | 6桁の確認コード |
| succeeded_at | timestamp | 確認成功日時（NULL=未確認） |
| created_at | timestamp | 作成日時 |
| updated_at | timestamp | 更新日時 |

**インデックス**:
- `email_confirmations_pkey` - PRIMARY KEY (id)
- `index_email_confirmations_on_created_at` - created_at
- `index_email_confirmations_on_email_and_code` - UNIQUE (email, code)

### コード設計

#### パッケージ構成

```
internal/
├── handler/
│   ├── password_reset/           # パスワードリセット開始
│   │   ├── handler.go            # Handler構造体
│   │   ├── new.go                # GET /password_reset
│   │   ├── create.go             # POST /password_reset
│   │   └── request.go            # リクエストバリデーション
│   │
│   ├── email_confirmation/       # メール確認
│   │   ├── handler.go            # Handler構造体
│   │   ├── new.go                # GET /email_confirmation
│   │   ├── create.go             # POST /email_confirmation
│   │   └── request.go            # リクエストバリデーション
│   │
│   └── password/                 # パスワード設定
│       ├── handler.go            # Handler構造体
│       ├── edit.go               # GET /password/edit
│       ├── update.go             # PATCH /password
│       └── request.go            # リクエストバリデーション
│
├── usecase/
│   ├── create_email_confirmation.go  # 確認コード生成・メール送信
│   ├── confirm_email.go              # 確認コード検証
│   └── update_password.go            # パスワード更新
│
├── repository/
│   └── email_confirmation_repository.go  # EmailConfirmationRepository
│
├── model/
│   └── email_confirmation.go         # EmailConfirmationモデル
│
├── query/queries/
│   └── email_confirmations.sql       # SQLクエリ定義
│
├── email/                            # メール送信
│   ├── sender.go                     # メール送信インターフェース
│   └── templates/                    # メールテンプレート
│       └── email_confirmation.templ
│
└── templates/pages/
    ├── password_reset/
    │   └── new.templ                 # メールアドレス入力フォーム
    │
    ├── email_confirmation/
    │   └── new.templ                 # 確認コード入力フォーム
    │
    └── password/
        └── edit.templ                # 新しいパスワード入力フォーム
```

#### 主要な構造体

**Handler構造体**

```go
// internal/handler/password_reset/handler.go
type Handler struct {
    cfg                   *config.Config
    sessionMgr            *session.Manager
    createEmailConfirmUC  *usecase.CreateEmailConfirmationUsecase
    turnstile             turnstile.Verifier
}

// internal/handler/email_confirmation/handler.go
type Handler struct {
    cfg                   *config.Config
    sessionMgr            *session.Manager
    emailConfirmRepo      *repository.EmailConfirmationRepository
    confirmEmailUC        *usecase.ConfirmEmailUsecase
}

// internal/handler/password/handler.go
type Handler struct {
    cfg                   *config.Config
    sessionMgr            *session.Manager
    emailConfirmRepo      *repository.EmailConfirmationRepository
    updatePasswordUC      *usecase.UpdatePasswordUsecase
}
```

**ユースケース**

```go
// internal/usecase/create_email_confirmation.go
type CreateEmailConfirmationInput struct {
    Email  string
    Event  string  // "password_reset"
    Locale string
}

type CreateEmailConfirmationResult struct {
    EmailConfirmation *model.EmailConfirmation
}

// internal/usecase/confirm_email.go
type ConfirmEmailInput struct {
    EmailConfirmationID string
    Code                string
}

type ConfirmEmailResult struct {
    EmailConfirmation *model.EmailConfirmation
}

// internal/usecase/update_password.go
type UpdatePasswordInput struct {
    Email    string
    Password string
}
```

**リポジトリ**

```go
// internal/repository/email_confirmation_repository.go
type EmailConfirmationRepository struct {
    db *sql.DB
    q  *query.Queries
}

func (r *EmailConfirmationRepository) Create(ctx context.Context, ec *model.EmailConfirmation) error
func (r *EmailConfirmationRepository) GetActiveByID(ctx context.Context, id string) (*model.EmailConfirmation, error)
func (r *EmailConfirmationRepository) GetSucceededByID(ctx context.Context, id string) (*model.EmailConfirmation, error)
func (r *EmailConfirmationRepository) MarkAsSucceeded(ctx context.Context, id string) error
```

### メール送信設計

Resend APIを使用してメールを送信します。

```go
// internal/email/sender.go
type Sender interface {
    SendEmailConfirmation(ctx context.Context, input SendEmailConfirmationInput) error
}

type SendEmailConfirmationInput struct {
    To     string
    Code   string
    Locale string
}
```

**メールテンプレート（日本語）**:
```
件名: [Mewst] 確認用コード

{email} さん、こんにちは。

確認用コードは下記になります。

{code}

確認用コードの有効期間は15分です。

もしこのメールに心当たりが無い場合は無視してください。

Mewst
https://mewst.com
```

### セッション管理

確認フローの状態管理にはセッションを使用します。

```go
// セッションに保存するキー
const (
    SessionKeyEmailConfirmationID = "email_confirmation_id"
)

// パスワードリセットフロー開始時
sessionMgr.SetValue(ctx, w, SessionKeyEmailConfirmationID, emailConfirmation.ID)

// パスワード更新後
sessionMgr.DeleteValue(ctx, w, SessionKeyEmailConfirmationID)
sessionMgr.ResetSession(ctx, w)  // 全セッション無効化
```

### バリデーション

**メールアドレス入力（CreateRequest）**:
- 必須チェック
- メールアドレス形式チェック

**確認コード入力（ConfirmRequest）**:
- 必須チェック
- 6桁の数字形式チェック
- コードの一致確認
- 有効期限チェック（15分）

**パスワード入力（UpdateRequest）**:
- 必須チェック
- 最小文字数チェック（8文字以上）
- 最大バイト数チェック（72バイト以下、bcrypt制限）

### テスト戦略

各コンポーネントに対してユニットテストと統合テストを実装します。

**ハンドラーテスト**:
- 正常系: フォーム表示、リダイレクト動作
- 異常系: バリデーションエラー、認証エラー
- 認証ガード: ログイン済みユーザーのリダイレクト

**ユースケーステスト**:
- 確認コード生成のランダム性
- メール送信の呼び出し
- トランザクション管理

**リポジトリテスト**:
- CRUD操作
- 有効期限フィルタリング

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

### フェーズ 1: インフラ準備（データベース・メール送信）

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
-->

- [x] **1-1**: SQLクエリとリポジトリの実装

  - `internal/query/queries/email_confirmations.sql` にSQLクエリを定義
  - `sqlc generate` でGoコードを生成
  - `internal/repository/email_confirmation_repository.go` を実装
  - `internal/model/email_confirmation.go` を実装
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 300 行（実装 200 行 + テスト 100 行）

- [x] **1-2**: メール送信機能の実装

  - `internal/email/sender.go` にSenderインターフェースと実装を追加
  - `internal/email/templates/email_confirmation.templ` にメールテンプレートを追加
  - Resend APIとの連携を実装
  - config.goに必要な環境変数を追加（MEWST_RESEND_API_KEY, MEWST_FROM_EMAIL）
  - **想定ファイル数**: 約 5 ファイル（実装 4 + テスト 1）
  - **想定行数**: 約 250 行（実装 180 行 + テスト 70 行）

### フェーズ 2: パスワードリセット開始機能

- [x] **2-1**: パスワードリセット開始ハンドラーの実装

  - `internal/handler/password_reset/handler.go` を実装
  - `internal/handler/password_reset/new.go` を実装（GET /password_reset）
  - `internal/handler/password_reset/create.go` を実装（POST /password_reset）
  - `internal/handler/password_reset/request.go` を実装
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 350 行（実装 250 行 + テスト 100 行）

- [x] **2-2**: パスワードリセットユースケースの実装

  - `internal/usecase/create_email_confirmation.go` を実装
  - トランザクション管理とメール送信を実装
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

- [x] **2-3**: パスワードリセットテンプレートの実装

  - `internal/templates/pages/password_reset/new.templ` を実装
  - Turnstileコンポーネントを組み込み
  - i18n翻訳キーを追加
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 200 行（実装 150 行 + テスト 50 行）

### フェーズ 3: メール確認機能

- [x] **3-1**: メール確認ハンドラーの実装

  - `internal/handler/email_confirmation/handler.go` を実装
  - `internal/handler/email_confirmation/new.go` を実装（GET /email_confirmation）
  - `internal/handler/email_confirmation/create.go` を実装（POST /email_confirmation）
  - `internal/handler/email_confirmation/request.go` を実装
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 350 行（実装 250 行 + テスト 100 行）

- [x] **3-2**: メール確認ユースケースの実装

  - `internal/usecase/confirm_email.go` を実装
  - 確認コード検証と成功マーク処理を実装
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 150 行（実装 80 行 + テスト 70 行）

- [x] **3-3**: メール確認テンプレートの実装

  - `internal/templates/pages/email_confirmation/new.templ` を実装
  - i18n翻訳キーを追加
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 150 行（実装 100 行 + テスト 50 行）

### フェーズ 4: パスワード更新機能

- [x] **4-1**: パスワード更新ハンドラーの実装

  - `internal/handler/password/handler.go` を実装
  - `internal/handler/password/edit.go` を実装（GET /password/edit）
  - `internal/handler/password/update.go` を実装（PATCH /password）
  - `internal/handler/password/request.go` を実装
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 350 行（実装 250 行 + テスト 100 行）

- [x] **4-2**: パスワード更新ユースケースの実装

  - `internal/usecase/update_password.go` を実装
  - bcryptでのハッシュ化とDB更新を実装
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 150 行（実装 80 行 + テスト 70 行）

- [x] **4-3**: パスワード更新テンプレートの実装

  - `internal/templates/pages/password/edit.templ` を実装
  - パスワード要件の表示
  - i18n翻訳キーを追加
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 150 行（実装 100 行 + テスト 50 行）

### フェーズ 5: 統合とルーティング

- [x] **5-1**: ルーティング設定と統合

  - `cmd/server/main.go` にルーティングを追加
  - ミドルウェアの適用（認証ガード、Method Override）
  - リバースプロキシのホワイトリストを更新
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 150 行（実装 100 行 + テスト 50 行）

- [ ] **5-2**: E2Eテストの実装

  - パスワードリセットフロー全体の統合テスト
  - 正常系・異常系のシナリオをカバー
  - **想定ファイル数**: 約 2 ファイル（実装 0 + テスト 2）
  - **想定行数**: 約 300 行（実装 0 行 + テスト 300 行）

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **レート制限**: 同一メールアドレスへの連続送信制限（将来的に検討）
- **確認コード再送機能**: コード再送ボタンの実装（将来的に検討）
- **パスワード変更履歴**: 過去N回と同じパスワードの禁止（将来的に検討）
- **二要素認証との連携**: 2FA設定済みユーザーへの追加確認（将来的に検討）

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Rails版パスワードリセット実装](/workspace/rails/app/controllers/password_resets/)
- [Rails版メール確認実装](/workspace/rails/app/controllers/email_confirmations/)
- [Go版ログイン機能実装](/workspace/go/internal/handler/sign_in/)
- [Resend Go SDK](https://github.com/resend/resend-go)
- [templ テンプレートガイド](/workspace/go/docs/templ-guide.md)
- [ハンドラーガイド](/workspace/go/docs/handler-guide.md)
