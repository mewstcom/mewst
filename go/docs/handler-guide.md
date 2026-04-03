# HTTP ハンドラーのガイドライン

このドキュメントは、Mewst Go 版で HTTP ハンドラーを実装する際のガイドラインを提供します。

## 目次

- [基本方針](#基本方針)
- [ディレクトリ構造](#ディレクトリ構造)
- [ファイル命名規則](#ファイル命名規則)
- [メソッド命名規則](#メソッド命名規則)
- [Handler 構造体の定義](#handler構造体の定義)
- [依存性注入のガイドライン](#依存性注入のガイドライン)
- [バリデーターの配置](#バリデーターの配置)
- [ルーティング登録](#ルーティング登録)
- [実装例](#実装例)

## 基本方針

HTTP ハンドラーは以下の原則に従って実装します：

- **リソースごとにディレクトリを切る**: すべてのエンドポイントはリソースディレクトリを作成
- **1 エンドポイント = 1 ハンドラーファイル**: 各エンドポイントは個別のファイルに実装
- **統一された命名規則**: ファイル名とメソッド名に一貫性を持たせる
- **例外なくディレクトリ化**: 単独のエンドポイントでも必ずディレクトリを作成

### なぜこの方針を採用するのか

- **可読性の向上**: ファイルサイズが小さく保たれ、コードが理解しやすい
- **保守性の向上**: 1 ファイル 1 責務の原則により、変更箇所が明確
- **拡張性の確保**: 新規機能追加時にどこに実装すべきか迷わない
- **並行開発の促進**: ファイル衝突が発生しにくい

## ディレクトリ構造

すべてのハンドラーは `internal/handler/` 配下にリソースディレクトリを作成します：

```
internal/handler/
├── popular_work/
│   ├── handler.go       # Handler構造体と依存性
│   ├── index.go         # Index メソッド (GET /works/popular)
│   └── index_test.go    # Index のテスト
├── password_reset/
│   ├── handler.go              # Handler構造体と依存性
│   ├── new.go                  # New (GET /password/reset) - リセット申請フォーム
│   ├── create.go               # Create (POST /password/reset)
│   └── handler_test.go         # ハンドラーの統合テスト
├── password/
│   ├── handler.go              # Handler構造体と依存性
│   ├── edit.go                 # Edit (GET /password/edit)
│   ├── update.go               # Update (PATCH /password)
│   └── handler_test.go         # ハンドラーの統合テスト
├── sign_in/
│   ├── handler.go              # Handler構造体と依存性
│   ├── new.go                  # New (GET /sign_in) - サインインフォーム
│   ├── create.go               # Create (POST /sign_in)
│   └── handler_test.go         # ハンドラーの統合テスト
├── health/
│   ├── handler.go       # Handler構造体と依存性
│   ├── show.go          # Show (GET /health) - ヘルスチェック
│   └── show_test.go     # Show のテスト
├── home/
│   ├── handler.go       # Handler構造体と依存性
│   ├── show.go          # Show (GET /) - ホームページ
│   └── show_test.go     # Show のテスト
└── manifest/
    ├── handler.go       # Handler構造体と依存性
    ├── show.go          # Show (GET /manifest.json) - PWAマニフェスト
    └── show_test.go     # Show のテスト
```

### リソースディレクトリの原則

- **すべてのエンドポイントをディレクトリ化**: 例外なく、すべてのエンドポイントはリソースディレクトリを作成
- **統一性**: 単独のエンドポイントでも必ずディレクトリを作成（例: `health/`, `home/`, `manifest/`）
- **拡張性**: 将来的にエンドポイントが追加されても容易に対応可能
- **リソース名**: 名詞として成立するリソース名を使用（例: `popular_work/`, `password_reset/`, `sign_in/`）

### リソース名の命名規則

- リソース名は**名詞**にする（例: `popular_work`, `password_reset`, `sign_in`）
- 形容詞+名詞の組み合わせの場合は英語の自然な語順にする（例: `popular_work` ⭕️、`work_popular` ❌）
- URL パターンから名詞を抽出してリソース名を決定

**命名例**:

| URL パターン    | リソース名（推奨）   | 理由                         |
| --------------- | -------------------- | ---------------------------- |
| /works/popular  | `popular_work/` ⭕️   | 形容詞+名詞の自然な語順      |
| /works/popular  | `work_popular/` ❌   | 「作品\_人気」は不自然       |
| /password/reset | `password_reset/` ⭕️ | 名詞として成立               |
| /users/me       | `current_user/` ⭕️   | 「現在のユーザー」という名詞 |
| /search         | `search/` ⭕️         | 「検索」という名詞           |

## ファイル命名規則

### 標準ファイル名（8 種類のみ）

リソースディレクトリ内には、以下の標準的なファイル名**のみ**を使用します：

- `handler.go` - Handler 構造体と依存性の定義
- `index.go` - 一覧ページ表示 (GET /resources)
- `show.go` - 個別ページ表示 (GET /resources/:id)
- `new.go` - 新規作成フォーム表示 (GET /resources/new)
- `create.go` - 作成処理 (POST /resources)
- `edit.go` - 編集フォーム表示 (GET /resources/:id/edit)
- `update.go` - 更新処理 (PATCH /resources/:id)
- `delete.go` - 削除処理 (DELETE /resources/:id)

バリデーションは `internal/validator/` パッケージに配置します。詳細は [@go/docs/validation-guide.md](validation-guide.md) を参照してください。

### 重要な原則

- 上記 8 種類以外のファイル名は**使用しない**
- 複雑な名前（`show_reset_form.go`, `process_reset.go` など）が必要な場合は、**新しいリソースディレクトリを作成する**
- 例: `password/show_reset_form.go` ではなく、`password_reset/new.go` を使用

### テストファイル

- **ハンドラーテスト**: `handler_test.go`（ハンドラー全体の統合テスト）
- **個別ハンドラーテスト**: `{action}_test.go`（例: `index_test.go`, `show_test.go`）も許可

バリデーションのテストは `internal/validator/` パッケージ内に配置します。

## メソッド命名規則

リソースディレクトリ内では以下のメソッド名を使用します：

- `Index` - 一覧ページ表示（ファイル名: `index.go`）
- `Show` - 個別ページ表示（ファイル名: `show.go`）
- `New` - 新規作成フォーム表示（ファイル名: `new.go`）
- `Create` - 作成処理（ファイル名: `create.go`）
- `Edit` - 編集フォーム表示（ファイル名: `edit.go`）
- `Update` - 更新処理（ファイル名: `update.go`）
- `Delete` - 削除処理（ファイル名: `delete.go`）

### ファイル名とメソッド名の対応

| HTTP メソッド | URL 例          | ファイル名  | メソッド名 | 説明         |
| ------------- | --------------- | ----------- | ---------- | ------------ |
| GET           | /works          | `index.go`  | `Index`    | 一覧取得     |
| GET           | /works/:id      | `show.go`   | `Show`     | 個別取得     |
| GET           | /works/new      | `new.go`    | `New`      | 新規フォーム |
| POST          | /works          | `create.go` | `Create`   | 作成処理     |
| GET           | /works/:id/edit | `edit.go`   | `Edit`     | 編集フォーム |
| PATCH         | /works/:id      | `update.go` | `Update`   | 更新処理     |
| DELETE        | /works/:id      | `delete.go` | `Delete`   | 削除処理     |

**ファイル名とメソッド名の完全一致**:

- すべてのファイル名とメソッド名が完全に対応しています（例: `index.go` → `Index` メソッド、`show.go` → `Show` メソッド）
- この一貫性により、コードの可読性と保守性が向上します
- HTTP メソッド（GET/POST/PATCH/DELETE）とハンドラーメソッド名（Index/Show/Create/Update/Delete）は明確に区別されています

**重要**: 複雑なメソッド名（`ShowResetForm`, `ProcessReset` など）は**使用しない**。代わりに新しいリソースディレクトリを作成し、標準的なメソッド名を使用します。

**例**:

- ❌ `password.ShowResetForm()` ではなく、✅ `password_reset.New()` を使用
- ❌ `password.ProcessReset()` ではなく、✅ `password_reset.Create()` を使用

### エンドポイントとリソースの対応例

#### 複数アクションを持つリソース

| エンドポイント       | リソースディレクトリ | ファイル名  | メソッド名 | 説明                           |
| -------------------- | -------------------- | ----------- | ---------- | ------------------------------ |
| GET /works/popular   | `popular_work/`      | `index.go`  | `Index`    | 人気作品一覧                   |
| GET /password/reset  | `password_reset/`    | `new.go`    | `New`      | パスワードリセット申請フォーム |
| POST /password/reset | `password_reset/`    | `create.go` | `Create`   | パスワードリセット申請処理     |
| GET /password/edit   | `password/`          | `edit.go`   | `Edit`     | パスワード変更フォーム         |
| PATCH /password      | `password/`          | `update.go` | `Update`   | パスワード変更処理             |
| GET /sign_in         | `sign_in/`           | `new.go`    | `New`      | サインインフォーム             |
| POST /sign_in        | `sign_in/`           | `create.go` | `Create`   | サインイン処理                 |

#### 単独エンドポイント（1 つのアクションのみ持つリソース）

| エンドポイント     | リソースディレクトリ | ファイル名 | メソッド名 | 説明             |
| ------------------ | -------------------- | ---------- | ---------- | ---------------- |
| GET /              | `home/`              | `show.go`  | `Show`     | ホームページ     |
| GET /health        | `health/`            | `show.go`  | `Show`     | ヘルスチェック   |
| GET /manifest.json | `manifest/`          | `show.go`  | `Show`     | PWA マニフェスト |

## Handler 構造体の定義

各リソースディレクトリの `handler.go` に Handler 構造体と依存性を定義します。

### 基本的な構造

```go
// handler/popular_work/handler.go
package popular_work

import (
    "github.com/mewstcom/mewst/internal/config"
    repository "github.com/mewstcom/mewst/internal/repository/sqlc"
)

// Handler は人気作品関連のHTTPハンドラーです
type Handler struct {
    cfg     *config.Config
    queries *repository.Queries
}

// NewHandler は新しいHandlerを作成します
func NewHandler(cfg *config.Config, queries *repository.Queries) *Handler {
    return &Handler{
        cfg:     cfg,
        queries: queries,
    }
}
```

### 命名規則

- **構造体名**: `Handler` （すべてのリソースで統一）
- **コンストラクタ名**: `NewHandler` （すべてのリソースで統一）
- **パッケージ名**: リソース名と同じ（例: `popular_work`, `password_reset`）

## 依存性注入のガイドライン

Handler 構造体には、そのリソースで必要な依存性を注入します。

### 基本方針

1. **共通の依存性**: すべてのエンドポイントで使う依存性は必ず含める
2. **専用の依存性**: 一部のエンドポイントでしか使わない依存性も含めて OK（多少の冗長性は許容）
3. **肥大化の防止**: Handler 構造体のフィールドが**8 個を超えたら**、リソース分割を検討
4. **段階的な設計**: 最初は必要最小限から始め、必要に応じて依存性を追加

### 許容される冗長性の例

```go
// ✅ 良い例: update.goでしか使わない依存性も含める（3-4個程度なら許容）
type Handler struct {
    cfg                   *config.Config
    queries               *repository.Queries
    db                    *sql.DB
    sessionMgr            *session.Manager
    updatePasswordUsecase *usecase.UpdatePasswordUsecase  // update.goでしか使わない
}
```

**理由**:

- 依存性が明示的で、このリソースが何に依存しているか一目でわかる
- ポインタなので未使用でもメモリオーバーヘッドは小さい（8 バイト程度）
- 将来的に他のエンドポイントでも使う可能性がある

### 肥大化の警告

```go
// ⚠️ 注意: フィールドが8個を超えたらリソース分割を検討
type Handler struct {
    cfg            *config.Config
    queries        *repository.Queries
    db             *sql.DB
    sessionMgr     *session.Manager
    limiter        *ratelimit.Limiter
    riverClient    *worker.Client
    createUsecase  *usecase.CreatePasswordUsecase
    updateUsecase  *usecase.UpdatePasswordUsecase
    resetUsecase   *usecase.ResetPasswordUsecase
    // ↑ このような場合は、リソースを更に細分化すべき
}
```

### リソース分割の例

依存性が多い場合は、以下のようにリソースを分割します：

```
# 分割前（肥大化）
password/
├── handler.go  # Handler構造体（10個の依存性）
├── edit.go
├── update.go
├── reset.go
└── confirm.go

# 分割後（適切なサイズ）
password_edit/
├── handler.go  # 2-3個の依存性
└── edit.go

password_update/
├── handler.go  # 2-3個の依存性
└── update.go

password_reset/
├── handler.go  # 2-3個の依存性
├── new.go
└── create.go
```

## バリデーターの配置

バリデーションは `internal/validator/` パッケージに配置します。`main.go` で構築し、UseCase に注入します。Handler は Validator を直接 import せず、すべて UseCase 経由でバリデーションを実行します。

- **構造体名**: `{Handler}{Action}Validator`（例: `SignInCreateValidator`, `PasswordUpdateValidator`）
- **戻り値**: Go の慣習に従った `(*Output, error)` の 2 値返し。バリデーション失敗時は `*model.ValidationError` を返す

### 配置例

```
internal/validator/
├── sign_in.go              # サインインのバリデーション（形式チェック + ユーザー検索、パスワード照合）
├── sign_in_test.go         # サインインのバリデーションテスト
├── password_reset.go       # パスワードリセットのバリデーション
├── password_reset_test.go  # パスワードリセットのバリデーションテスト
├── password.go             # パスワード更新のバリデーション
└── password_test.go        # パスワード更新のバリデーションテスト
```

詳細なバリデーション実装については、[@go/docs/validation-guide.md](validation-guide.md) を参照してください。

## ルーティング登録

`cmd/server/main.go` でルーティングを登録する際は、リソースごとにハンドラーを初期化します。

### 基本的な登録方法

```go
// Validatorの構築（main.goで構築し、UseCaseに注入）
signInValidator := validator.NewSignInCreateValidator(userRepo)
passwordResetValidator := validator.NewPasswordResetCreateValidator()
passwordValidator := validator.NewPasswordUpdateValidator()

// UseCaseの初期化（Validatorを注入）
createSignInUC := usecase.NewCreateSignInUsecase(db, sessionRepo, signInValidator)
createPasswordResetUC := usecase.NewCreatePasswordResetUsecase(db, userRepo, passwordResetValidator)
updatePasswordUC := usecase.NewUpdatePasswordUsecase(db, userRepo, passwordValidator)

// ハンドラーの初期化（UseCaseのみを注入）
popularWorkHandler := popular_work.NewHandler(cfg, getPopularWorksUC)
passwordResetHandler := password_reset.NewHandler(cfg, sessionMgr, createPasswordResetUC)
passwordHandler := password.NewHandler(cfg, sessionMgr, updatePasswordUC)
signInHandler := sign_in.NewHandler(cfg, sessionMgr, createSignInUC)

// ルーティング登録
r.Get("/works/popular", popularWorkHandler.Index)
r.Get("/password/reset", passwordResetHandler.New)
r.Post("/password/reset", passwordResetHandler.Create)
r.Get("/password/edit", passwordHandler.Edit)
r.Patch("/password", passwordHandler.Update)
r.Get("/sign_in", signInHandler.New)
r.Post("/sign_in", signInHandler.Create)
```

### ルーティングの原則

- **標準的な HTTP メソッドを使用**: GET/POST/PATCH/DELETE
- **更新処理は PATCH を使用**: 部分更新を表現し、Rails との整合性も保つ
- **Method Override パターン**: HTML フォームから PATCH/DELETE を実現（詳細は [@go/CLAUDE.md](../CLAUDE.md#httpメソッドとルーティング) を参照）

## 実装例

### 例 1: 単独エンドポイント（health/）

```go
// internal/handler/health/handler.go
package health

import (
    "github.com/mewstcom/mewst/internal/config"
)

// Handler はヘルスチェック関連のHTTPハンドラーです
type Handler struct {
    cfg *config.Config
}

// NewHandler は新しいHandlerを作成します
func NewHandler(cfg *config.Config) *Handler {
    return &Handler{
        cfg: cfg,
    }
}
```

```go
// internal/handler/health/show.go
package health

import (
    "net/http"
)

// Show GET /health - ヘルスチェック
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
}
```

### 例 2: 複数エンドポイント（password_reset/）

```go
// internal/handler/password_reset/handler.go
package password_reset

import (
    "github.com/mewstcom/mewst/internal/config"
    "github.com/mewstcom/mewst/internal/session"
    "github.com/mewstcom/mewst/internal/usecase"
)

// Handler はパスワードリセット関連のHTTPハンドラーです
type Handler struct {
    cfg                    *config.Config
    sessionMgr             *session.Manager
    createPasswordResetUC  *usecase.CreatePasswordResetUsecase  // UseCase のみに依存
}

// NewHandler は新しいHandlerを作成します
func NewHandler(
    cfg *config.Config,
    sessionMgr *session.Manager,
    createPasswordResetUC *usecase.CreatePasswordResetUsecase,
) *Handler {
    return &Handler{
        cfg:                   cfg,
        sessionMgr:            sessionMgr,
        createPasswordResetUC: createPasswordResetUC,
    }
}
```

```go
// internal/handler/password_reset/new.go
package password_reset

import (
    "net/http"
    "github.com/mewstcom/mewst/internal/templates/pages/password_reset"
)

// New GET /password/reset - パスワードリセット申請フォーム
func (h *Handler) New(w http.ResponseWriter, r *http.Request) {
    // 実装
}
```

```go
// internal/handler/password_reset/create.go
package password_reset

import (
    "net/http"
)

// Create POST /password/reset - パスワードリセット申請処理
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    // 実装
}
```

## 設計思想

この命名規則は、以下の設計思想に基づいています：

- **RESTful 設計**: Rails をはじめ多くの Web フレームワークが採用する標準的な 7 アクション（index, show, new, create, edit, update, delete）
- **完全な対応**: ファイル名とメソッド名が 100%一致
- **HTTP メソッドとの区別**: HTTP メソッド（GET/POST/PATCH/DELETE）とハンドラーメソッド名（Index/Show/Create/Update/Delete）は明確に区別
- **Rails からの移行に最適**: 既存の Rails 開発者が即座に理解できる命名
- **MPA 向け最適化**: HTML レンダリング型 Web アプリケーションに適した命名

## まとめ

HTTP ハンドラーを実装する際は、以下のポイントを守ってください：

1. **すべてのエンドポイントをディレクトリ化**: 例外なく、リソースディレクトリを作成
2. **標準的なファイル名を使用**: 8 種類のファイル名のみを使用
   - `handler.go`, `index.go`, `show.go`, `new.go`, `create.go`, `edit.go`, `update.go`, `delete.go`
3. **ファイル名とメソッド名を一致させる**: 可読性と保守性を向上
4. **依存性注入を適切に管理**: 8 個以下のフィールドを目安に、必要に応じてリソースを分割
5. **バリデーションは `internal/validator/` パッケージに配置**: `main.go` で構築して UseCase に注入（Handler は Validator に直接依存しない）

これらの規則を守ることで、以下のメリットが得られます：

- コードの可読性と保守性が向上する
- 新規機能追加時に迷わない
- 並行開発がスムーズになる
- テストが書きやすくなる

## 関連ドキュメント

- [@go/CLAUDE.md](../CLAUDE.md) - Go 版開発ガイド
- [@go/docs/validation-guide.md](validation-guide.md) - リクエストバリデーションガイド
- [@go/docs/templ-guide.md](templ-guide.md) - templ テンプレートガイド
- [@go/docs/architecture-guide.md](architecture-guide.md) - アーキテクチャガイド
