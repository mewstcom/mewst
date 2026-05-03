# アーキテクチャガイド

このドキュメントは、Go 版 Mewst のアーキテクチャパターンを説明します。

## 概要

Go 版 Mewst は、関心の分離を意識した**3 層アーキテクチャ**を採用しています。

### 3 層アーキテクチャの構成

```
┌─────────────────────────────────────────────────────────┐
│ Presentation層（プレゼンテーション層）                    │
│ - Handler, Worker, Email                               │
│ - ViewModel                                            │
│ - Template                                             │
│ - Middleware                                           │
│ - Presentation層のヘルパー（i18n, session）                │
└─────────────────────────────────────────────────────────┘
         ↓ 依存（OK）
┌─────────────────────────────────────────────────────────┐
│ Application層（アプリケーション層）                        │
│ - UseCase, Validator                                  │
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

- **ファイル名**: スネークケース（`user_calendar.go`）。`_repository.go` のような suffix は付けない(`{Model}.go ↔ repository/{model}.go` の 1:1 対応を維持する)
- **構造体名**: パスカルケース（`UserCalendar`, `UserCalendarRepository`）
- **Model と Repository は同じ名前**: `model/user_calendar.go` ↔ `repository/user_calendar.go`

#### Not Found の表現

Repository の取得系メソッド(`FindByID` 等)は、対象が存在しない場合に `(nil, nil)` を返します。`sql.ErrNoRows` を独自エラー(`ErrNotFound` 等)に変換する設計は採用しません。

```go
func (r *UserRepository) FindByID(ctx context.Context, id model.UserID) (*model.User, error) {
    row, err := r.q.GetUserByID(ctx, uuid.UUID(id))
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return r.toModel(row), nil
}
```

呼び出し側(UseCase / Validator / Session Manager 等)は `if user == nil` で未存在を判定します。「未存在」を業務上の異常として扱う場合は、UseCase 側で `*model.AppError`(`AppErrCodeResourceNotFound`)や `usecase.ErrNotFound` に変換して上位層へ伝搬します。

#### モデルの重複を避ける

クエリの結果や状態ごとに新しいモデルを作らず、既存のモデルを再利用します。関連エンティティのデータが必要な場合は、ポインタ型のフィールドでモデル間の参照を表現します。

```go
// ✅ 良い例: 既存の Work モデルに User への参照を持たせる
type Work struct {
    ID    int64
    User  *User  // 関連エンティティへのポインタ参照
    Title string
}

// ❌ 悪い例: クエリ結果に合わせた専用モデルを作る
type JoinedWork struct {
    WorkID    int64
    WorkTitle string
    UserID    int64
    UserName  string
}
```

Repository ではクエリ結果ごとに変換メソッドを用意し、同じモデルに変換します：

```go
// 単純なクエリ結果 → Work（User は ID のみ）
func (r *WorkRepository) toModel(row query.Work) *model.Work { ... }

// JOINクエリ結果 → Work（User のフィールドをより多く設定）
func (r *WorkRepository) toWorksFromJoinedRows(rows []query.ListJoinedWorksByUserRow) []*model.Work { ... }
```

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
4. **UseCase** (Application 層): Repository 経由で Model を取得し、ビジネスロジックを実行して結果を返す
5. **Handler** (Presentation 層): UseCase を呼び出して Model を取得し、Model を ViewModel に変換
6. **ViewModel** (Presentation 層): 表示用のデータ構造（画像 URL 生成、言語切り替えなど）
7. **Template** (Presentation 層): ViewModel を受け取って HTML を生成

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
- **internal/validator**: バリデーション（形式チェック + DB を使った状態検証を統合。`main.go` で構築し UseCase に注入）

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

### Presentation 層（Handler, Worker, Email, ViewModel, Template, Middleware）

各パッケージの依存関係：

- **Templates**: `ViewModel` を通じてデータを表示。データアクセス（`repository`, `query`）、ビジネスロジック（`usecase`）、`Model` への直接依存は禁止。
- **ViewModel**: `Model` → `ViewModel` の変換のみ。`repository`, `query` に依存しない
- **Handler**: `query`, `repository`, `validator` への直接アクセス禁止。すべて `usecase` を経由する
- **Worker**: UseCase を呼ぶだけの薄い Adapter。`query`, `repository`, `handler`, `middleware`, `viewmodel`, `templates` に依存しない
- **Email**: テンプレートレンダリング + API 送信を内包。`templates`, `i18n` に依存可能。上位層（`handler`, `middleware`, `usecase`, `validator`, `worker`, `dispatcher`）およびデータアクセス層（`query`, `repository`）、`viewmodel`, `session` には依存しない
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

### Application 層（UseCase, Validator）

- **UseCase**: `query` への直接アクセス禁止。データアクセスは `repository` を経由。`session` への直接アクセス禁止。Presentation 層（`handler`, `middleware`, `viewmodel`, `templates`, `worker`）に依存しない。ジョブのキュー投入は `dispatcher` を経由する。Validator を統合し、バリデーション → 永続化をオーケストレーション（詳細は[UseCaseオーケストレーション](#usecaseオーケストレーション)を参照）
- **Validator**: 形式チェック + 状態バリデーションを統合。`repository`, `model` に依存可能。`query` への直接アクセスは禁止（Repository を経由）。`usecase` には依存しない（UseCase が Validator を呼び出す方向）。Presentation 層（`handler`, `middleware`, `viewmodel`, `templates`）に依存しない（翻訳は `i18n.T()` を直接使用）

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

ビジネスロジックとトランザクション管理は `internal/usecase` パッケージで行います。Handler / Worker からのすべてのデータアクセス・認可・バリデーション・永続化は UseCase を経由します。

📖 **詳細な命名規則・実装パターン・処理順序・WithTx パターン・テスト方針については [@go/docs/usecase-guide.md](usecase-guide.md) を参照してください。**

### UseCase の種類

Handler はすべてのデータアクセスを UseCase 経由で行う。UseCase は以下の 3 種類に分類される。

| 種類                         | 責務                                                            | Validator | トランザクション                |
| ---------------------------- | --------------------------------------------------------------- | --------- | ------------------------------- |
| 読み取り UseCase             | データ取得、複数 Repository の集約                              | なし      | なし                            |
| 書き込み UseCase             | 永続化処理 (作成・更新・削除)、ビジネスロジック                 | なし      | あり/なし (必要に応じて WithTx) |
| オーケストレーション UseCase | Validator を統合し、フォーム送信のバリデーション → 永続化を統括 | あり      | あり/なし (必要に応じて WithTx) |

オーケストレーション UseCase は書き込み UseCase の特殊形だが、Validator 統合の有無で実装パターンが大きく変わるため独立カテゴリとして扱う。

### UseCase オーケストレーション

Handler は Validator に直接依存せず、UseCase がバリデーション → 永続化を統括する。

```
Handler → UseCase.Execute(input)
            ↓
          Validator.Validate(input)
            ↓ エラー時: *model.ValidationError を返す
          Repository（永続化処理）
            ↓
          Handler ← (*Output, error)
```

Handler は `model.AsValidationError` / `model.AsAppError` でエラー種別を判別する。具体的なエラーハンドリングは [@go/docs/handler-guide.md](handler-guide.md) を参照。

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

Validator は Go の慣習に従った `(data, error)` の 2 値返しで、データを返さない場合は `error` のみを返す。バリデーション失敗時は `*model.ValidationError`（`error` を満たす）を返す。詳細は [validation-guide.md](validation-guide.md#構造体の命名規則) を参照。

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

`AppErrorCode` の現行定数:

| 定数                         | 用途                           |
| ---------------------------- | ------------------------------ |
| `AppErrCodeResourceNotFound` | リソース未存在(404 相当)       |
| `AppErrCodeForbidden`        | 権限不足(403 相当)             |
| `AppErrCodeConflict`         | 状態の競合(409 相当)           |
| `AppErrCodeInternal`         | 想定済みの内部エラー(500 相当) |

`AppError` の生成は構造体リテラルで行う(コンストラクタ関数は用意しない):

```go
return &model.AppError{
    Code:    model.AppErrCodeResourceNotFound,
    UserMsg: i18n.T(ctx, "error_not_found_message"),
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
