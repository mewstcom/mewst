# アーキテクチャガイド

このドキュメントは、Go 版 Mewst のアーキテクチャパターンを説明します。

## 概要

Go 版 Mewst は、関心の分離を意識した**3 層アーキテクチャ**を採用しています。

### 3 層アーキテクチャの構成

```
┌─────────────────────────────────────────────────────────┐
│ Presentation層（プレゼンテーション層）                    │
│ - Handler, Worker, Email                               │
│ - Validator                                            │
│ - ViewModel                                            │
│ - Template                                             │
│ - Middleware                                           │
│ - Presentation層のヘルパー（i18n, session）                │
└─────────────────────────────────────────────────────────┘
         ↓ 依存（OK）
┌─────────────────────────────────────────────────────────┐
│ Application層（アプリケーション層）                        │
│ - UseCase（ビジネスフロー、トランザクション管理、          │
│   バリデーション統合）                                    │
└─────────────────────────────────────────────────────────┘
         ↓ 依存（OK）
┌─────────────────────────────────────────────────────────┐
│ Domain/Infrastructure層（統合）                          │
│ - Query (sqlc), Repository, Model                      │
│ - Dispatcher                                           │
│ （同じ層なので相互に依存できる）                          │
└─────────────────────────────────────────────────────────┘
```

**重要**: Domain/Infrastructure 層は Presentation 層に**依存してはいけない**

### Domain/Infrastructure 層を統合する理由

このプロジェクトでは、Domain 層と Infrastructure 層を分離せず、統合して扱います：

- **実用的**: データベース変更（PostgreSQL → MySQL など）はほぼ起こらない
- **シンプル**: 層をまたぐ変換コストを削減し、シンプルさを保つ
- **Go らしい**: Go のプラグマティックな哲学に合致
- **YAGNI 原則**: 必要になってから層を分ければ良い

Repository と Model を同じ層として扱うことで、依存関係がシンプルになり、相互に依存できます。

### Model と Repository の 1:1 関係

各ドメインエンティティに対して対応する Repository を作成します：

- `model.Work` ↔ `repository.WorkRepository`
- `model.User` ↔ `repository.UserRepository`
- `model.Episode` ↔ `repository.EpisodeRepository`

この 1:1 関係により、以下のメリットがあります：

- **一貫性**: どの Model に対してどの Repository を使うかが明確
- **保守性**: Model の変更が Repository に集約される
- **可読性**: コードの見通しが良くなる

#### 命名規則

Model と Repository のファイル名・構造体名は統一します：

| Model          | Repository               | ファイル名         |
| -------------- | ------------------------ | ------------------ |
| `Work`         | `WorkRepository`         | `work.go`          |
| `User`         | `UserRepository`         | `user.go`          |
| `UserCalendar` | `UserCalendarRepository` | `user_calendar.go` |

**命名のルール**:

- **ファイル名**: スネークケース（`user_calendar.go`）
- **構造体名**: パスカルケース（`UserCalendar`, `UserCalendarRepository`）
- **Model と Repository は同じ名前**: `model/user_calendar.go` ↔ `repository/user_calendar.go`

#### Query ファイルの命名

Query ファイルは用途に応じて 2 つのパターンがあります：

**1. テーブル名ベース（単純な CRUD 操作）**:

- 単一テーブルに対する CRUD 操作
- 例: `users.sql`, `works.sql`, `sessions.sql`

**2. モデル/機能名ベース（複雑なクエリ）**:

- 複数テーブルを JOIN するクエリ
- 特定のモデルを構築するためのクエリ
- 例: `user_calendar.sql`（users, library_entries, slots, works を JOIN）

```
internal/query/queries/
├── users.sql           # usersテーブルのCRUD
├── works.sql           # worksテーブルのCRUD
├── sessions.sql        # sessionsテーブルのCRUD
└── user_calendar.sql   # UserCalendarモデル用の複合クエリ
```

### データの流れ

1. **Query** (Domain/Infrastructure 層): SQL クエリを実行し、クエリ結果（`query.GetPopularWorksRow`など）を返す
2. **Repository** (Domain/Infrastructure 層): Query 結果を Model に変換し、複数のクエリを組み合わせる
3. **Model** (Domain/Infrastructure 層): ページに依存しない汎用的なドメインエンティティ（`model.Work`など）
4. **Handler** (Presentation 層): UseCase から Model を取得し、Model を ViewModel に変換
5. **ViewModel** (Presentation 層): 表示用のデータ構造（画像 URL 生成、言語切り替えなど）
6. **Template** (Presentation 層): ViewModel を受け取って HTML を生成

### 主要なレイヤー

- **Handler**: HTTP リクエスト・レスポンスの処理
- **UseCase**: ビジネスロジックとトランザクション管理
- **ViewModel**: プレゼンテーション層のデータ変換
- **Repository**: Query 結果を Model に変換
- **Model**: ドメインエンティティ
- **Query**: sqlc 生成コード（データアクセス層）

## レイヤーごとのパッケージ分類

### Presentation 層（プレゼンテーション層）

- **internal/handler**: HTTP リクエストハンドラー（リソースごとにディレクトリを切り、1 エンドポイント = 1 ファイルの原則）
- **internal/viewmodel**: プレゼンテーション層のデータ変換（View 用のデータ構造）
- **internal/templates**: templ テンプレートファイル（型安全な HTML テンプレート）
  - `layouts/`: レイアウトテンプレート（`default.templ`, `simple.templ`）
  - `components/`: 再利用可能なコンポーネント（`head.templ`, `flash.templ`, `form_errors.templ`）
  - `pages/`: ページテンプレート（機能別にディレクトリを分割）
  - `emails/`: メールテンプレート
  - `helper.go`: テンプレートヘルパー関数（`T()`, `Locale()`, `Icon()`など）
- **internal/middleware**: HTTP ミドルウェア
  - `reverse_proxy.go`: Rails 版へのリバースプロキシミドルウェア（Go 版で未実装の機能を Rails 版にプロキシ）
  - `auth.go`: 認証ミドルウェア
  - `csrf.go`: CSRF 保護ミドルウェア
  - `method_override.go`: HTTP メソッドオーバーライドミドルウェア
- **internal/validator**: バリデーション（形式チェック + DB を使った状態検証を統合。`main.go` で構築し UseCase に注入）
- **internal/worker**: バックグラウンドジョブの実行（river ベース）
  - UseCase を呼ぶだけの薄い Adapter として実装
- **internal/email**: メール送信（テンプレートレンダリング + Resend API 送信）
  - `sender.go`: メール送信の基盤（Resend API 連携）
  - `confirmation.go`: メール確認コード送信（テンプレートレンダリング + i18n を内包）
  - `templates`, `i18n` に依存可能。`handler`, `usecase`, `worker` には依存しない

**Presentation 層のヘルパー**（Presentation 層内で使用可能）:

- **internal/i18n**: 国際化（翻訳取得、言語切り替え）
- **internal/session**: セッション管理（フラッシュメッセージ、ユーザー情報）

### Application 層（アプリケーション層）

- **internal/usecase**: ビジネスロジック層（フラットなファイル配置）
  - ビジネスフロー、トランザクション管理を担当
  - 複数の Repository を組み合わせた処理
  - Validator を統合し、バリデーション → 永続化をオーケストレーション

### Domain/Infrastructure 層（統合）

- **internal/query**: sqlc で生成されるコード（旧 `repository/sqlc`）
  - `queries/`: SQL クエリファイル（sqlc で処理）
  - 単一の SQL クエリを実行する責務のみ
  - 手動編集禁止
- **internal/model**: ドメインモデル
  - ページに依存しない汎用的なドメインエンティティ（`Work`, `Cast`, `Staff`など）
  - Presentation 層に依存しない（`image.Helper`などに依存しない）
- **internal/repository**: Repository 層
  - Query 結果を Model に変換する
  - 複数のクエリを組み合わせて Model を構築
  - **Model と Repository は 1:1 の関係**（例: `model.Work` ↔ `repository.WorkRepository`）

- **internal/dispatcher**: ジョブキューへの投入を抽象化（UseCase ↔ Worker 間の循環依存を解消）
  - Args 型の定義と Enqueue メソッドを提供
  - 依存先は river（外部ライブラリ）のみ

### その他

- **cmd/server/main.go**: エントリポイント。設定、データベース接続、Chi ルーターを使用した HTTP サーバーを初期化
- **internal/config**: 環境変数から設定を読み込む設定管理。`.env.{environment}` ファイルを使用
- **internal/auth**: 認証ロジック（セキュアトークン生成、パスワードハッシュ）
- **internal/turnstile**: Cloudflare Turnstile 連携
- **internal/ratelimit**: レートリミット（`query` への直接依存は禁止、`repository` を経由）
- **internal/database**: データベース接続管理
- **internal/clientip**: クライアント IP アドレス検出

## レイヤー間の依存関係

### 基本方針

```
Presentation層 → Application層 → Domain/Infrastructure層
```

下位層は上位層に依存しません（依存の方向は一方通行）。

### Presentation 層（Handler, Worker, Email, Validator, ViewModel, Template, Middleware）

各パッケージの依存関係：

- **Templates**: `ViewModel` を通じてデータを表示。データアクセス（`repository`, `query`）、ビジネスロジック（`usecase`）、`Model` への直接依存は禁止。
- **ViewModel**: `Model` → `ViewModel` の変換のみ。`repository`, `query` に依存しない
- **Handler**: `query`, `repository`, `validator` への直接アクセス禁止。すべて `usecase` を経由する
- **Worker**: UseCase を呼ぶだけの薄い Adapter。`query`, `handler`, `middleware`, `viewmodel`, `templates` に依存しない
- **Email**: テンプレートレンダリング + API 送信を内包。`templates`, `i18n` に依存可能。`handler`, `usecase`, `worker`, `validator`, `dispatcher`, `session` には依存しない
- **Validator**: 形式チェック + 状態バリデーションを統合。`repository`, `model` に依存可能。`query` への直接アクセスは禁止（Repository を経由）。`usecase` には依存しない（UseCase が Validator を呼び出す方向）
- **Middleware**: エラーページ・メンテナンスページ等のレンダリングのため `templates` への依存は許可。`query`, `repository`, `usecase`, `handler`, `viewmodel` に依存しない

**依存関係の図解**:

```
Templates → ViewModel (OK: 表示用データを受け取る)
              ↓
ViewModel → Model (OK: ドメインデータを表示用に変換)
              ↓
Handler → UseCase, ViewModel
Worker  → UseCase, Dispatcher
Email   → Templates, i18n (OK: テンプレートレンダリング)
  ↑
Middleware → Templates (OK: エラーページ等のレンダリング)
```

**重要**: Templates は ViewModel に依存できますが、Model に直接依存することは禁止です。必ず ViewModel を経由してください。

### Application 層（UseCase）

- `query` への直接アクセス禁止。データアクセスは `repository` を経由
- `session` への直接アクセス禁止。session は Presentation 層のヘルパー
- Presentation 層（`handler`, `middleware`, `viewmodel`, `templates`）に依存しない
- Validator を統合し、バリデーション → 永続化をオーケストレーション（詳細は[UseCaseオーケストレーション](#usecaseオーケストレーション)を参照）

### Domain/Infrastructure 層（Query, Repository, Model）

各パッケージの依存関係：

- **Model**: 純粋なドメインエンティティ。`query`, `repository` に依存しない
- **Repository**: `query`, `model` に依存できる。上位層に依存しない
- **Query**: sqlc 生成コード。他のすべての層に依存しない（独立）

**依存関係の図解**:

```
Query (独立、単独で動作)
  ↓
Repository → Query, Model
  ↓
Model (独立、他に依存しない)
```

**重要**: Repository が Query の結果を Model に変換します。

### 重要なルール

1. **Query への依存は Repository のみ**: Handler/UseCase が Query に直接依存することは禁止
2. **Handler は UseCase のみを経由**: Handler は Repository、Validator に直接依存せず、すべて UseCase 経由で行う
3. **UseCase が Validator を統合**: バリデーションは UseCase 内で実行し、`*model.ValidationError` として Handler に返す
4. **Worker は薄い Adapter**: UseCase を呼ぶだけの実装にし、ビジネスロジックを持たない
5. **下位層は上位層に依存しない**: Domain/Infrastructure 層は Presentation 層に依存しない
6. **関心の分離**: 各パッケージは明確な責務を持ち、その責務に集中する

**依存関係の強制**: これらのルールは `.golangci.yml` の depguard 設定により静的解析レベルで強制されています。`make lint` で違反を検出できます。

### なぜ Repository のみが Query に依存すべきか

**メリット**:

- ✅ **保守性**: データアクセスロジックが Repository に集約される
- ✅ **拡張性**: キャッシュ層の追加、データソース変更が Repository のみで完結
- ✅ **一貫性**: 「データ取得 = Repository を使う」というルールが明確
- ✅ **テスト容易性**: Repository をモックすれば、Handler/UseCase のテストが容易

**デメリットを回避**:

- ❌ データアクセスロジックの散在（Handler/UseCase に直接 Query を書く）
- ❌ 変更の波及（データアクセス方法の変更が Handler/UseCase に影響）
- ❌ ルールの曖昧さ（「このケースは Query を直接使って良い？」という混乱）

## 物理的な構造と論理的な構造

- **物理的な構造**: `internal/`配下はフラット（機能別にパッケージを分ける）
- **論理的な構造**: ドキュメントでレイヤーごとにパッケージを分類し、依存関係を明示

## ビューモデル（View Model）

### 概要

プレゼンテーション層でのデータ変換は `internal/viewmodel` パッケージで行います。

### 責務

- リポジトリ層のデータ構造をテンプレート表示用の構造に変換
- 画像 URL 生成、日付フォーマットなどの表示ロジック
- 複数のリポジトリ結果を組み合わせた表示用データの作成

### 命名規則

- **構造体名**: `Work`, `User` など（エンティティ名と同じ）
- **変換関数**: `NewWorkFromXXX` （XXX は sqlc が生成した型名）
- **複数変換**: `NewWorksFromXXX` （複数形）

### 利点

- ハンドラーをシンプルに保つ
- データ変換ロジックの再利用が可能
- テストしやすい構造
- sqlc が生成する型とプレゼンテーション層の分離

### 実装例

```go
// internal/viewmodel/work.go
package viewmodel

import (
    "github.com/mewstcom/mewst/internal/repository"
)

// Work はテンプレートで表示する作品データ
type Work struct {
    ID            int64
    Title         string
    ImageURL      string
    WatchersCount int32
    SeasonYear    *int32
    SeasonName    *string
}

// NewWorkFromPopularRow は人気作品クエリの結果をViewModelに変換
func NewWorkFromPopularRow(cfg *config.Config, work repository.GetPopularWorksRow) Work {
    return Work{
        ID:            work.ID,
        Title:         work.Title,
        ImageURL:      generateImageURL(cfg, work.ImageData),
        WatchersCount: work.WatchersCount,
        SeasonYear:    work.SeasonYear,
        SeasonName:    work.SeasonName,
    }
}

// NewWorksFromPopularRows は複数の作品を一括変換
func NewWorksFromPopularRows(cfg *config.Config, works []repository.GetPopularWorksRow) []Work {
    result := make([]Work, len(works))
    for i, work := range works {
        result[i] = NewWorkFromPopularRow(cfg, work)
    }
    return result
}

// generateImageURL は画像データからimgproxy経由のURLを生成
func generateImageURL(cfg *config.Config, imageData *string) string {
    if imageData == nil {
        return ""
    }
    // imgproxy URLを生成
    // ...
}
```

### ハンドラーでの使用

```go
// internal/handler/popular_works.go
package handler

import (
    "github.com/mewstcom/mewst/internal/templates/layouts"
    "github.com/mewstcom/mewst/internal/templates/pages/works"
    "github.com/mewstcom/mewst/internal/viewmodel"
)

func (h *Handler) PopularWorks(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // リポジトリから作品データを取得
    worksRows, err := h.queries.GetPopularWorks(ctx)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // ViewModelに変換
    worksView := viewmodel.NewWorksFromPopularRows(h.cfg, worksRows)

    // ページメタデータを作成
    meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
    meta.SetTitle(ctx, "popular_anime_title")

    // テンプレートをレンダリング
    user := authMiddleware.GetUserFromContext(ctx)
    layouts.Default(ctx, meta, user, works.Popular(ctx, worksView)).Render(ctx, w)
}
```

### テスト

```go
func TestNewWorkFromPopularRow(t *testing.T) {
    cfg := &config.Config{
        Domain: "example.com",
    }

    work := repository.GetPopularWorksRow{
        ID:            1,
        Title:         "テストアニメ",
        WatchersCount: 100,
        ImageData:     stringPtr(`{"id":"test.jpg"}`),
    }

    result := viewmodel.NewWorkFromPopularRow(cfg, work)

    if result.ID != 1 {
        t.Errorf("ID = %d, want 1", result.ID)
    }
    if result.Title != "テストアニメ" {
        t.Errorf("Title = %q, want %q", result.Title, "テストアニメ")
    }
    if result.WatchersCount != 100 {
        t.Errorf("WatchersCount = %d, want 100", result.WatchersCount)
    }
    if result.ImageURL == "" {
        t.Error("ImageURL should not be empty")
    }
}
```

## ユースケース（Use Case）

### 概要

ビジネスロジックとトランザクション管理は `internal/usecase` パッケージで行います。

### UseCase の種類

Handler はすべてのデータアクセスを UseCase 経由で行う。UseCase は以下の 3 種類に分類される：

| 種類                         | 責務                                            | トランザクション        |
| ---------------------------- | ----------------------------------------------- | ----------------------- |
| オーケストレーション UseCase | バリデーション → 永続化の統合（フォーム送信系） | あり（WithTx パターン） |
| 書き込み UseCase             | 永続化処理、ビジネスロジック                    | あり（WithTx パターン） |
| 読み取り UseCase             | データ取得、複数 Repository の集約              | なし                    |

**書き込み UseCase**:

- トランザクションを伴う永続化処理（作成・更新・削除）
- 複数の Repository を跨ぐビジネスロジック
- ロールバックが必要な複合操作

```go
// 例: StripeサブスクライバーとUserを同時に更新する場合
type DeleteStripeSubscriberUsecase struct {
    db                   *sql.DB
    stripeSubscriberRepo *repository.StripeSubscriberRepository
    userRepo             *repository.UserRepository
}

func (uc *DeleteStripeSubscriberUsecase) Execute(ctx context.Context, input Input) (*Result, error) {
    tx, err := uc.db.BeginTx(ctx, nil)
    // トランザクション内で複数のRepositoryを操作
}
```

**読み取り UseCase**:

- Handler が必要とするデータ取得
- 複数の Repository を組み合わせたデータ集約
- トランザクション不要な参照処理

```go
// 例: メール確認データの取得（読み取り専用）
type GetActiveEmailConfirmationUsecase struct {
    emailConfirmationRepo *repository.EmailConfirmationRepository
}

func NewGetActiveEmailConfirmationUsecase(
    emailConfirmationRepo *repository.EmailConfirmationRepository,
) *GetActiveEmailConfirmationUsecase {
    return &GetActiveEmailConfirmationUsecase{
        emailConfirmationRepo: emailConfirmationRepo,
    }
}

type GetActiveEmailConfirmationInput struct {
    ID uuid.UUID
}

type GetActiveEmailConfirmationOutput struct {
    EmailConfirmation *model.EmailConfirmation
}

func (uc *GetActiveEmailConfirmationUsecase) Execute(ctx context.Context, input GetActiveEmailConfirmationInput) (*GetActiveEmailConfirmationOutput, error) {
    ec, err := uc.emailConfirmationRepo.GetActiveByID(ctx, input.ID)
    if err != nil {
        return nil, fmt.Errorf("メール確認の取得に失敗: %w", err)
    }
    return &GetActiveEmailConfirmationOutput{EmailConfirmation: ec}, nil
}
```

### Repository の WithTx パターン

Usecase でトランザクションを使用する場合、Repository の `WithTx` メソッドを使ってトランザクション内で操作を行います。

#### 目的

- Repository をコンストラクタで受け取り、`Execute` 内で `WithTx(tx)` を呼び出すことで、トランザクション内で安全にデータ操作を行う
- 同じ Repository インターフェースを、通常時（DB 直接）とトランザクション時（tx 経由）の両方で使用できる

#### メリット

- **トランザクション境界の明確化**: `BeginTx` から `Commit` までのスコープが Usecase 内で完結する
- **ロールバック安全性**: `defer func() { _ = tx.Rollback() }()` により、エラー時に確実にロールバックされる
- **Repository の再利用**: 同じ Repository を通常の読み取りとトランザクション内の書き込みの両方で使用できる

#### 実装例

```go
type CreateAccountUsecase struct {
    db          *sql.DB
    userRepo    *repository.UserRepository
    profileRepo *repository.ProfileRepository
}

func NewCreateAccountUsecase(
    db *sql.DB,
    userRepo *repository.UserRepository,
    profileRepo *repository.ProfileRepository,
) *CreateAccountUsecase {
    return &CreateAccountUsecase{
        db:          db,
        userRepo:    userRepo,
        profileRepo: profileRepo,
    }
}

func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
    tx, err := uc.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    // トランザクション内で操作するためのリポジトリを取得
    userRepo := uc.userRepo.WithTx(tx)
    profileRepo := uc.profileRepo.WithTx(tx)

    // ユーザーを作成
    user, err := userRepo.Create(ctx, input.Email, input.PasswordDigest)
    if err != nil {
        return nil, fmt.Errorf("ユーザーの作成に失敗: %w", err)
    }

    // プロフィールを作成
    _, err = profileRepo.Create(ctx, user.ID, input.Atname)
    if err != nil {
        return nil, fmt.Errorf("プロフィールの作成に失敗: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
    }

    return &CreateAccountOutput{UserID: user.ID}, nil
}
```

#### 重要なポイント

- **`defer func() { _ = tx.Rollback() }()`**: `Commit` 成功後の `Rollback` は no-op なので、常に defer で呼び出して安全にロールバックを保証する
- **`WithTx(tx)` の呼び出しタイミング**: `BeginTx` の後、実際のデータ操作の前に呼び出す
- **元の Repository は変更されない**: `WithTx` は新しいインスタンスを返すため、元の Repository はトランザクションに影響されない

### 責務

- トランザクション管理（`db.BeginTx` から `tx.Commit` まで）
- 複数の repository を跨ぐ処理
- ビジネスロジックの実装

### ファイル配置

`internal/usecase/` 直下にフラットに配置（サブディレクトリは作成しない）

### 命名規則

- **ファイル名**: `{action}_{entity}.go`
  - 例: `create_session.go`, `create_password_reset_token.go`, `update_password_reset.go`
  - **重要**: 動詞（アクション）を必ず先頭に配置する
- **構造体名**: `{Action}{Entity}Usecase`
  - 例: `CreateSessionUsecase`, `CreatePasswordResetTokenUsecase`
  - 注: `Usecase` の `c` は小文字（既存コードとの統一のため）
- **コンストラクタ**: `New{Action}{Entity}Usecase`
- **Execute メソッド**: `Execute(ctx context.Context, ...) (*Result, error)`

### 結果型

各 UseCase は専用の Result 構造体を返します。

例: `SessionResult`, `CreatePasswordResetTokenResult`

### 利点

- ハンドラーがシンプルになる（HTTP 処理に専念できる）
- トランザクション境界が明確
- テストしやすい構造
- ビジネスロジックの再利用が可能

### 実装例

#### シンプルなユースケース（トランザクションなし）

```go
// internal/usecase/create_session.go
package usecase

import (
    "context"
    "github.com/mewstcom/mewst/internal/repository"
)

type CreateSessionUsecase struct {
    queries *repository.Queries
}

func NewCreateSessionUsecase(queries *repository.Queries) *CreateSessionUsecase {
    return &CreateSessionUsecase{queries: queries}
}

type SessionResult struct {
    PublicID string
    UserID   int64
}

func (uc *CreateSessionUsecase) Execute(ctx context.Context, userID int64) (*SessionResult, error) {
    // セッションIDを生成
    publicID := generateSecureRandomString(32)

    // セッションをDBに保存
    session, err := uc.queries.CreateSession(ctx, repository.CreateSessionParams{
        PublicID: publicID,
        UserID:   userID,
    })
    if err != nil {
        return nil, fmt.Errorf("セッションの作成に失敗: %w", err)
    }

    return &SessionResult{
        PublicID: session.PublicID,
        UserID:   session.UserID,
    }, nil
}
```

#### 複雑なユースケース（トランザクションあり）

```go
// internal/usecase/create_password_reset_token.go
package usecase

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    "github.com/mewstcom/mewst/internal/repository"
)

type CreatePasswordResetTokenUsecase struct {
    db      *sql.DB
    queries *repository.Queries
}

func NewCreatePasswordResetTokenUsecase(db *sql.DB, queries *repository.Queries) *CreatePasswordResetTokenUsecase {
    return &CreatePasswordResetTokenUsecase{
        db:      db,
        queries: queries,
    }
}

type CreatePasswordResetTokenResult struct {
    Token  string
    UserID int64
}

func (uc *CreatePasswordResetTokenUsecase) Execute(ctx context.Context, userID int64) (*CreatePasswordResetTokenResult, error) {
    // トランザクション開始
    tx, err := uc.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
    }
    defer tx.Rollback()

    // トランザクション対応のクエリを作成
    qtx := uc.queries.WithTx(tx)

    // 既存のトークンを削除
    err = qtx.DeletePasswordResetTokensByUserID(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("既存トークンの削除に失敗: %w", err)
    }

    // 新しいトークンを生成
    token := generateSecureRandomString(32)
    hashedToken := hashToken(token)

    // トークンをDBに保存
    expiresAt := time.Now().Add(24 * time.Hour)
    _, err = qtx.CreatePasswordResetToken(ctx, repository.CreatePasswordResetTokenParams{
        UserID:      userID,
        Token:       hashedToken,
        ExpiresAt:   expiresAt,
    })
    if err != nil {
        return nil, fmt.Errorf("トークンの作成に失敗: %w", err)
    }

    // トランザクションをコミット
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
    }

    return &CreatePasswordResetTokenResult{
        Token:  token,
        UserID: userID,
    }, nil
}
```

### ハンドラーでの使用

```go
// internal/handler/password_reset.go
package handler

import (
    "github.com/mewstcom/mewst/internal/usecase"
)

func (h *Handler) ProcessPasswordReset(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // リクエストバリデーション
    req := &PasswordResetRequest{
        Email: r.FormValue("email"),
    }
    if formErrors := req.Validate(ctx); formErrors != nil {
        // エラー処理
        return
    }

    // ユーザーを検索
    user, err := h.queries.GetUserByEmail(ctx, req.Email)
    if err != nil {
        // ユーザーが見つからない場合の処理
        return
    }

    // ユースケースを実行
    uc := usecase.NewCreatePasswordResetTokenUsecase(h.db, h.queries)
    result, err := uc.Execute(ctx, user.ID)
    if err != nil {
        slog.ErrorContext(ctx, "トークンの作成に失敗", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // メール送信
    err = h.sendPasswordResetEmail(ctx, user.Email, result.Token)
    if err != nil {
        slog.ErrorContext(ctx, "メール送信に失敗", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // 成功レスポンス
    http.Redirect(w, r, "/password/reset_sent", http.StatusSeeOther)
}
```

### テスト

```go
func TestCreatePasswordResetTokenUsecase_Execute(t *testing.T) {
    // テストDBとトランザクションをセットアップ
    db, tx := testutil.SetupTx(t)
    queries := repository.New(db).WithTx(tx)

    // テストユーザーを作成
    userID := testutil.NewUserBuilder(t, tx).
        WithEmail("test@example.com").
        Build()

    // ユースケースを実行
    uc := usecase.NewCreatePasswordResetTokenUsecase(db, queries)
    result, err := uc.Execute(context.Background(), userID)

    // アサーション
    if err != nil {
        t.Fatalf("Execute() error = %v", err)
    }

    if result.Token == "" {
        t.Error("Token should not be empty")
    }

    if result.UserID != userID {
        t.Errorf("UserID = %d, want %d", result.UserID, userID)
    }

    // トークンがDBに保存されているか確認
    tokens, err := queries.GetPasswordResetTokensByUserID(context.Background(), userID)
    if err != nil {
        t.Fatalf("GetPasswordResetTokensByUserID() error = %v", err)
    }

    if len(tokens) != 1 {
        t.Errorf("len(tokens) = %d, want 1", len(tokens))
    }
}
```

### 命名の注意点

#### ファイル名の順序

`{action}_{entity}` の順（動詞を必ず先頭に）

- ✅ `create_session.go`
- ❌ `session_create.go`
- ✅ `create_password_reset_token.go`
- ❌ `password_reset_create_token.go`

#### 複合エンティティ

エンティティが複数単語の場合はアンダースコアで連結

- ✅ `create_password_reset_token.go` （password_reset_token というエンティティ）

#### 構造体名の大文字化

`Usecase` の `c` は小文字

- ✅ `CreateSessionUsecase`
- ❌ `CreateSessionUseCase`

### ファイル配置の理由

#### フラット構造

エンティティごとにディレクトリを作らず、`internal/usecase/` 直下にすべてのファイルを配置

理由:

- **検索性**: ファイル名のプレフィックスでグルーピングされるため、エディタで検索しやすい
- **シンプルさ**: ディレクトリ階層が深くならず、import パスがシンプル
- **スケーラビリティ**: ファイル数が増えても管理しやすい

## UseCase オーケストレーション

Handler は Validator に直接依存せず、UseCase がバリデーション → 永続化を統括する。

### フロー

```
Handler → UseCase.Execute(input)
            ↓
          Validator.Validate(input)
            ↓ エラー時: *model.ValidationError を返す
          Repository（永続化処理）
            ↓
          Handler ← (*Output, error)
```

### 実装例

```go
// UseCase がバリデーションを統合
func (uc *CreateSignInUsecase) Execute(ctx context.Context, input CreateSignInInput) (*CreateSignInOutput, error) {
    // 1. バリデーション（Validator を内部で呼び出し）
    validateOutput, err := uc.signInValidator.Validate(ctx, validator.SignInCreateValidatorInput{
        Email:    input.Email,
        Password: input.Password,
    })
    if err != nil {
        return nil, err // *model.ValidationError または素の error
    }

    // 2. ビジネスロジック（セッション作成等）
    // ...
    return &CreateSignInOutput{...}, nil
}
```

### Handler でのエラー判別

Handler は `model.AsValidationError` / `model.AsAppError` でエラー種別を判別する。

```go
output, err := h.signInUC.Execute(ctx, input)
if err != nil {
    if ve := model.AsValidationError(err); ve != nil {
        // フォームを再表示（422 Unprocessable Entity）
        w.WriteHeader(http.StatusUnprocessableEntity)
        h.renderForm(w, r, ve, ...)
        return
    }
    if ae := model.AsAppError(err); ae != nil {
        slog.ErrorContext(ctx, ae.LogString())
        // ae.Code に応じた HTTP ステータスコードを返す
    }
    // 予期しないエラー → 500
    slog.ErrorContext(ctx, "予期しないエラー", "error", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
}
```

## エラー型

`internal/model/errors.go` に定義されたエラー型を使用してレイヤー間のエラー伝搬を行う。

### エラー型の使い分け

| エラー型                 | 生成元    | 意味                             | Handler の対応                          |
| ------------------------ | --------- | -------------------------------- | --------------------------------------- |
| `*model.ValidationError` | Validator | 入力が不正（ユーザーが修正可能） | フォーム再描画（422）                   |
| `*model.AppError`        | UseCase   | 業務レベルの既知の失敗           | エラーコードに応じた処理（403, 404 等） |
| 素の `error`             | どこでも  | 予期しないシステムエラー         | 500                                     |

### ValidationError

バリデーションエラー。Handler はフォームを再描画する。

```go
type ValidationError struct {
    Global []string            // フォーム全体のエラー
    Fields map[string][]string // フィールドごとのエラー
}
```

Validator は `(*ValidatorOutput, error)` の 2 値を返し、バリデーション失敗時は `*model.ValidationError`（`error` を満たす）を返す。

### AppError（SafeError パターン）

アプリケーションエラー。`Error()` はユーザー安全なメッセージのみを返し、内部エラーの露出を構造的に防止する。

```go
type AppError struct {
    Code     AppErrorCode      // Handler がステータスコードを決定するために使用
    UserMsg  string            // ユーザーに表示する安全なメッセージ
    Internal error             // ログ出力用の内部エラー
    Metadata map[string]string // 構造化ログ用のメタデータ
}
```

### ヘルパー関数

```go
// エラーから特定の型を取り出す（errors.As のラッパー）
model.AsValidationError(err) *ValidationError // nil ならその型ではない
model.AsAppError(err) *AppError               // nil ならその型ではない
```

## Worker と Dispatcher

### Worker（Presentation 層）

バックグラウンドジョブを実行する薄い Adapter。UseCase を呼ぶだけの実装にし、ビジネスロジックを持たない。

```go
// Worker は UseCase を呼ぶだけ
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[...]) error {
    return w.uc.Execute(ctx, usecase.SendEmailConfirmationInput{...})
}
```

### Dispatcher（Domain/Infrastructure 層）

ジョブキューへの投入を抽象化し、UseCase ↔ Worker 間の循環依存を解消する。

```
依存の方向:
Worker (Presentation)     → dispatcher + usecase
UseCase (Application)     → dispatcher
Dispatcher (Domain/Infra) → river（外部ライブラリのみ）
```

UseCase がジョブをエンキューする場合は Dispatcher 経由で行い、Worker を直接 import しない。

## ベストプラクティス

### 1. ViewModel と UseCase の使い分け

```go
// ❌ Bad: ハンドラーで複雑な変換ロジック
func (h *Handler) PopularWorks(w http.ResponseWriter, r *http.Request) {
    works, _ := h.queries.GetPopularWorks(ctx)

    // ハンドラーで複雑な変換を行う（悪い例）
    worksView := make([]WorkView, len(works))
    for i, work := range works {
        imageURL := ""
        if work.ImageData != nil {
            // 複雑な画像URL生成ロジック
            imageURL = generateComplexImageURL(work.ImageData)
        }
        worksView[i] = WorkView{
            ID:       work.ID,
            Title:    work.Title,
            ImageURL: imageURL,
        }
    }
    // ...
}

// ✅ Good: ViewModelで変換
func (h *Handler) PopularWorks(w http.ResponseWriter, r *http.Request) {
    works, _ := h.queries.GetPopularWorks(ctx)
    worksView := viewmodel.NewWorksFromPopularRows(h.cfg, works)
    // ...
}
```

### 2. トランザクション管理は UseCase で

```go
// ❌ Bad: ハンドラーでトランザクション管理
func (h *Handler) CreateWork(w http.ResponseWriter, r *http.Request) {
    tx, _ := h.db.BeginTx(ctx, nil)
    defer tx.Rollback()

    // 複雑なビジネスロジック
    // ...

    tx.Commit()
}

// ✅ Good: UseCaseでトランザクション管理
func (h *Handler) CreateWork(w http.ResponseWriter, r *http.Request) {
    uc := usecase.NewCreateWorkUsecase(h.db, h.queries)
    result, err := uc.Execute(ctx, params)
    // ...
}
```

### 3. ハンドラーは HTTP 処理に専念

```go
// ✅ Good: ハンドラーは UseCase を呼び、エラー型で分岐する
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. フォームデータの取得
    email := r.FormValue("email")
    password := r.FormValue("password")

    // 2. UseCase を実行（バリデーション込み）
    output, err := h.signInUC.Execute(ctx, usecase.CreateSignInInput{
        Email: email, Password: password,
    })
    if err != nil {
        if ve := model.AsValidationError(err); ve != nil {
            w.WriteHeader(http.StatusUnprocessableEntity)
            h.renderForm(w, r, ve, email)
            return
        }
        slog.ErrorContext(ctx, "処理に失敗", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // 3. レスポンス
    http.Redirect(w, r, "/", http.StatusFound)
}
```

## 採用しなかった方針

### Handler → Repository を引き続き許可する（現状維持）

Handler が Repository を直接呼び出すことを許可し、規約とコードレビューで書き込み呼び出しを防止する方針。

**不採用の理由**:

- 規約だけでは書き込みメソッドの呼び出しを防止できない
- Handler の依存先が UseCase と Repository の 2 つに分散し、ルールが複雑になる
- 依存グラフが一方向に統一されず、アーキテクチャの見通しが悪い

### Repository を ReadRepository と WriteRepository に分離する

Repository をインターフェースで分離し、Handler には ReadRepository のみ注入する方針。

**不採用の理由**:

- インターフェースの管理コストが増える
- Go のプラグマティックな哲学に反する（過度な抽象化）
- UseCase 経由に統一するほうがルールとしてシンプル

### Validator を handler パッケージ内に残す

Validator を `internal/handler/` 内に配置したまま、depguard による強制は諦めて構造的な規約とコードレビューで対応する方針。

**不採用の理由**:

- depguard で強制できないと、違反が再発する可能性がある
- 「Handler パッケージは repository を import しない」というルールを完全に強制できるメリットが大きい
- Validator を独立パッケージにすることで、UseCase からバリデーションを再利用できる

### Handler が Validator を直接呼び出す

Handler が `internal/validator/` を直接 import してバリデーションを実行し、結果に応じて UseCase を呼び出す方針。

**不採用の理由**:

- Handler の責務が「HTTP 処理」と「バリデーション呼び出し」の 2 つに分散する
- UseCase にバリデーションを統合することで、Handler は UseCase のみに依存するシンプルな構造になる
- depguard で handler → validator の依存を禁止できるため、ルールが静的に強制される
- Worker からも UseCase を経由してバリデーション付きの処理を再利用できる

### ValidationError と AppError を Application 層に配置する

`internal/usecase/errors.go` または新設の `internal/apperror/` にエラー型を定義する方針。

**不採用の理由**:

- Validator が `ValidationError` を生成するために `usecase` パッケージを import すると、UseCase → Validator の依存方向に対して Validator → UseCase の逆方向依存が発生し、循環依存のリスクがある
- 新設パッケージ（`internal/apperror/`）を作ると、パッケージが増えて複雑になる
- Model（Domain/Infrastructure 層）は依存グラフの最下層にあり、すべての層から自然に参照できるため、エラー型の配置先として適切

### UseCase が templates を直接 import してメールをレンダリングする

UseCase（Application 層）が `internal/templates`（Presentation 層）を直接 import してメールテンプレートをレンダリングする方針。

**不採用の理由**:

- UseCase が Presentation 層に依存するのは、レイヤー間の依存方向に反する
- email パッケージに型固有の Sender（`ConfirmationSender`）を新設し、テンプレートレンダリングを email パッケージ側に閉じ込めることで、UseCase の `templates` 依存を回避できる
- UseCase は自前で定義した小さい interface（`EmailConfirmationSender`）に依存するだけでよく、よりシンプルになる

### session.GenerateToken() を repository に移動する

セッショントークン生成を Repository に移動する方針。

**不採用の理由**:

- Repository はデータアクセスの抽象化であり、トークン生成は Repository の責務ではない
- セッショントークンの生成は認証ロジックの一部であり、`internal/auth` パッケージが適切な配置先

## まとめ

- **ViewModel**: リポジトリ層のデータをテンプレート表示用に変換
- **UseCase**: ビジネスロジック、トランザクション管理、バリデーション統合（オーケストレーション）
- **Handler**: HTTP 処理に専念し、UseCase のみを呼び出して `errors.As` でエラー種別を判別
- **Worker**: バックグラウンドジョブの実行。UseCase を呼ぶだけの薄い Adapter
- **Dispatcher**: ジョブキューへの投入を抽象化し、UseCase ↔ Worker 間の循環依存を解消
- **エラー型**: `ValidationError`（フォーム再描画）と `AppError`（SafeError パターン）で型安全なエラー伝搬

この構造により、コードの見通しが良く、テストしやすく、保守しやすいアーキテクチャを実現できます。
