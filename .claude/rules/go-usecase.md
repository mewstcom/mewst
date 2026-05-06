---
paths:
  - "go/**/*.{go,templ}"
---

# ユースケースガイド

このドキュメントは、Go 版プロジェクトの UseCase 層の設計と実装パターンを説明します。

## 概要

UseCase はアプリケーションのオーケストレーターです。Handler/Worker からのすべてのデータアクセス・認可・バリデーション・ビジネスロジック・永続化は UseCase を経由します。

## 責務

- **オーケストレーション**: 認可チェック・バリデーション・ビジネスロジック・永続化を統括する
- データ取得ロジックの集約（読み取り UseCase）
- トランザクション管理（書き込み UseCase: `db.BeginTx` から `tx.Commit` まで）
- 複数の repository を跨ぐ処理

## UseCase の役割

Handler/Worker からのすべてのデータアクセスは UseCase を経由する。UseCase は以下の 3 種類に分類される。

### UseCase の分類

| 種類                         | 責務                                                            | Validator | トランザクション                |
| ---------------------------- | --------------------------------------------------------------- | --------- | ------------------------------- |
| 読み取り UseCase             | データ取得、複数 Repository の集約                              | なし      | なし                            |
| 書き込み UseCase             | 永続化処理 (作成・更新・削除)、ビジネスロジック                 | なし      | あり/なし (必要に応じて WithTx) |
| オーケストレーション UseCase | Validator を統合し、フォーム送信のバリデーション → 永続化を統括 | あり      | あり/なし (必要に応じて WithTx) |

オーケストレーション UseCase は書き込み UseCase の特殊形だが、Validator 統合の有無で実装パターンが大きく変わるため独立カテゴリとして扱う。

書き込み UseCase / オーケストレーション UseCase は、複数の永続化を跨ぐ場合やロールバックが必要な複合操作のときのみトランザクションを開く。単一の永続化呼び出しで完結する場合はトランザクションを伴わない。

### 各分類の役割

**読み取り UseCase**:

- Handler が必要とするデータ取得ロジックを集約する
- 複数の Repository を組み合わせてデータを取得する
- トランザクションは不要

**書き込み UseCase**:

- 永続化処理 (作成・更新・削除) とビジネスロジック
- 複数の Repository を跨ぐ場合や、ロールバックが必要な複合操作ではトランザクションを開く
- 単一の永続化呼び出しで完結する場合はトランザクションを伴わない (例: `CreateSessionUsecase` のように単一の永続化で完結する UseCase)
- **書き込み UseCase のルール**(詳細は「UseCase 内の処理順序」を参照):
  1. トランザクション開始後はデータの取得や計算処理を行わない。永続化処理のみ行う(トランザクション前のデータ取得は許可)
  2. Execute 内にロジックを直接書かない。ロジックは関数やメソッドとして定義し、Execute 内ではそれを呼び出すだけにする

**オーケストレーション UseCase**:

- 書き込み UseCase の中でも、Validator を統合してフォーム送信全体を統括するもの
- Handler は HTTP の入出力変換に徹し、認可・バリデーション・ビジネスロジック・永続化は UseCase 内部で完結する
- 書き込み UseCase のルールに加えて、Validator の戻り値 (`*model.ValidationError` または取得済みデータ) を扱う

### どの分類にすべきかの判断フロー

新しい UseCase を追加するときは、以下の順で判断する。

1. Repository の取得系メソッド (`Find*`) のみを呼ぶ?
   → **読み取り UseCase** (プレフィックス: `Get`)
2. Validator を統合する? (フォーム送信の入口を担う)
   → **オーケストレーション UseCase**
3. 上記以外 (永続化を伴うが Validator なし)
   → **書き込み UseCase**
   - 例: `CreateSessionUsecase` のように、Handler 側で先にバリデーションが終わっている前提でセッションを作成するだけの UseCase。または別の UseCase の内部から呼ばれる純粋な永続化 UseCase

### 実装パターンの例

書き込み UseCase の例:

```go
// Wikino の例: ページとスペースメンバーを同時に更新する書き込み UseCase
type CreatePageUsecase struct {
    db              *sql.DB
    pageRepo        *repository.PageRepository
    spaceMemberRepo *repository.SpaceMemberRepository
}

func (uc *CreatePageUsecase) Execute(ctx context.Context, input Input) (*Result, error) {
    tx, err := uc.db.BeginTx(ctx, nil)
    // トランザクション内で複数のRepositoryを操作
}
```

オーケストレーション UseCase の例:

```go
// Wikino の例: Validator を統合してフォーム送信を統括するオーケストレーション UseCase
type CreateSuggestionUsecase struct {
    db             *sql.DB
    validator      *validator.SuggestionCreateValidator
    suggestionRepo *repository.SuggestionRepository
}

func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input CreateSuggestionInput) (*CreateSuggestionOutput, error) {
    // Validator を統合: 失敗時は *model.ValidationError を返す
    if _, err := uc.validator.Validate(ctx, validatorInput); err != nil {
        return nil, err
    }
    // バリデーション成功後に永続化
    return uc.createSuggestion(ctx, input)
}
```

読み取り UseCase の例:

```go
// Wikino の例: トピック詳細ページのデータを集約する読み取り UseCase
type GetTopicDetailUsecase struct {
    spaceRepo       *repository.SpaceRepository
    spaceMemberRepo *repository.SpaceMemberRepository
    topicRepo       *repository.TopicRepository
    topicMemberRepo *repository.TopicMemberRepository
    pageRepo        *repository.PageRepository
}

type GetTopicDetailInput struct {
    SpaceIdentifier model.SpaceIdentifier
    TopicNumber     int32
    UserID          *model.UserID
    Page            int32
}

type GetTopicDetailOutput struct {
    Space       *model.Space
    SpaceMember *model.SpaceMember
    Topic       *model.Topic
    TopicMember *model.TopicMember
    PinnedPages []*model.Page
    Pages       []*model.Page
    Pagination  *repository.PaginationResult
}

func (uc *GetTopicDetailUsecase) Execute(ctx context.Context, input GetTopicDetailInput) (*GetTopicDetailOutput, error) {
    space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
    if err != nil {
        return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
    }
    // 複数のRepositoryからデータを取得して集約
    // ...
}
```

## ファイル配置

`internal/usecase/` 直下にフラットに配置（サブディレクトリは作成しない）

**プライベート関数の配置ルール**: あるUseCaseファイルに定義されたプライベート関数を別のUseCaseファイルから呼び出す必要が生じた場合は、その関数を専用のファイルに切り出す。ファイル名は関数の責務を表す名詞にする（例: Wikiリンク関連の共通関数を `linked_page.go` に配置）。

## 命名規則

- **ファイル名**: `{action}_{entity}.go`
  - 例: `create_session.go`, `create_password_reset_token.go`, `update_password_reset.go`
  - **重要**: 動詞（アクション）を必ず先頭に配置する
- **構造体名**: `{Action}{Entity}Usecase`
  - 例: `CreateSessionUsecase`, `CreatePasswordResetTokenUsecase`
  - 注: `Usecase` の `c` は小文字（既存コードとの統一のため）
- **読み取り UseCase のプレフィックスは `Get` に統一**: 読み取り UseCase のアクションには `Get` を使用する。`List` や `Fetch` など他の動詞は使用しない。これにより `Get` = 読み取り、それ以外 = 書き込みという判別が即座にできる
  - 例: `GetTopicDetailUsecase`, `GetPageDetailUsecase`, `GetDraftPagesUsecase`
- **コンストラクタ**: `New{Action}{Entity}Usecase`
- **Execute メソッド**: `Execute(ctx context.Context, ...) (*Result, error)`

## 結果型

各 UseCase は専用の Result 構造体を返します。

例: `SessionResult`, `CreatePasswordResetTokenResult`

## 利点

- ハンドラーがシンプルになる（HTTP 処理に専念できる）
- トランザクション境界が明確
- テストしやすい構造
- ビジネスロジックの再利用が可能

## UseCase 内の処理順序

書き込み UseCase は以下の順序で処理を実行する。

```
書き込み UseCase (オーケストレーター)
  1. データ取得（トランザクション外）
  2. 認可チェック（Policy）
  3. バリデーション（Validator）
  4. ビジネスロジック（計算、変換等）
  5. トランザクション（永続化のみ）
```

### 書き込み UseCase のルール

書き込み UseCase は以下の 2 つのルールを守る:

1. **トランザクション開始後はデータの取得や計算処理を行わない**: トランザクション内は永続化処理のみ行う。データ取得は `BeginTx` の前で済ませ、`auth.HashPassword` (bcrypt) のような重い CPU 計算もトランザクション開始前に実行する。トランザクションを長時間保持しないことで、ロックや競合の発生を抑える
2. **Execute をオーケストレーションに専念させる**: 永続化を含む実処理はプライベート関数 (またはメソッド) として定義し、Execute 内ではそれを呼び出すだけにする。Execute は「データ取得・認可・バリデーション・永続化関数の呼び出し」のオーケストレーションのみに専念する

   **例外**: ルール 2 の主旨は「Execute をオーケストレーションに専念させる」ことであり、オーケストレーションすべき対象がない場合は本ルールの適用範囲外となる。具体的には、単一の永続化呼び出し + 必要最小限の前処理だけで完結する書き込み UseCase は、プライベート関数化を省略して Execute に直接書いてよい。例: パスワード更新 UseCase のように「バリデーション → ハッシュ化 → UPDATE」だけで他にオーケストレーションするものがない場合、Execute 内で完結させる

   **関数切り出しが必要かどうかの判断フロー**:

   ```
   1. トランザクション内で永続化処理が 2 つ以上ある?
      → 必要 (プライベート関数化)
      例: 複数リソース (Profile / User / UserProfile / Actor 等) を 1 トランザクションで作成する UseCase

   2. 永続化前にロジック (計算・変換) が 2 行以上ある?
      → 必要 (プライベート関数化)
      例: 確認コード生成 + 永続化を行う UseCase

   3. 上記以外 (バリデーション → 1 つの永続化、または最小限の変換 → 1 つの永続化)
      → 不要 (Execute 直書きで OK)
      例: パスワード更新 UseCase (バリデーション → ハッシュ化 → 1 回の UPDATE)
   ```

   閾値 (「2 つ以上」「2 行以上」) は機械的に守るべき絶対値ではなく、「Execute がオーケストレーターとしての役割を持つかどうか」を見極めるための目安として扱う。境界例ではプロジェクトに既存する近い UseCase の書き方に合わせる

```go
// Wikino の例
// ✅ 良い例: 書き込み UseCase がデータ取得・認可・バリデーション・永続化を統括する
func (uc *CreateSuggestionUsecase) Execute(ctx context.Context, input CreateSuggestionInput) (*CreateSuggestionOutput, error) {
    // 1. データ取得（トランザクション外）
    space, err := uc.spaceRepo.FindByIdentifier(ctx, input.SpaceIdentifier)
    if err != nil {
        return nil, fmt.Errorf("スペースの取得に失敗: %w", err)
    }
    // 未存在を業務上の異常として扱う場合は AppError に変換 ((nil, nil) パターンの判定方法)
    if space == nil {
        return nil, &model.AppError{
            Code:    model.AppErrCodeResourceNotFound,
            UserMsg: i18n.T(ctx, "error_space_not_found"),
        }
    }

    // 2. 認可チェック
    if !policy.NewTopicPolicy(spaceMember, topicMember).CanCreateSuggestion() {
        return nil, &model.AppError{
            Code:    model.AppErrCodeForbidden,
            UserMsg: i18n.T(ctx, "error_forbidden"),
        }
    }

    // 3. バリデーション
    draftPages, err := uc.validator.Validate(ctx, validatorInput)
    if err != nil {
        return nil, err  // *model.ValidationError か素の error がそのまま上がる
    }

    // 4. ビジネスロジック + 5. トランザクション（永続化のみ）
    return uc.createSuggestion(ctx, input, draftPages)
}

// ❌ 悪い例: トランザクション内でデータ取得や CPU 計算を行っている
func (uc *WriteUsecase) Execute(ctx context.Context, input Input) error {
    tx, err := uc.db.BeginTx(ctx, nil)
    // トランザクション内でデータ取得 → トランザクション前に行うべき
    page, err := pageRepo.FindByID(ctx, input.PageID, input.SpaceID)
    // ...
    // bcrypt のハッシュ化もトランザクション内で実行 → トランザクション前に行うべき
    digest, err := auth.HashPassword(input.Password)
    // ...
}
```

```go
// Mewst の例: bcrypt のような重い CPU 計算をトランザクション開始前に済ませてから永続化関数を呼ぶ
import (
    "context"
    "fmt"
    "time"

    "example.com/app/internal/auth"
    "example.com/app/internal/validator"
)

func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
    // 1. バリデーション (トランザクション外)
    if err := uc.accountValidator.Validate(ctx, validator.AccountCreateValidatorInput{
        Email:    input.Email,
        Atname:   input.Atname,
        Password: input.Password,
    }); err != nil {
        return nil, err
    }

    // 2. CPU 計算 (bcrypt) と時刻取得をトランザクション外で済ませる (ルール 1)。
    // bcrypt はコスト 10 で 100ms 級の処理になるため、トランザクション内で実行すると
    // その間 DB 接続を専有してロック競合の原因になる。
    passwordDigest, err := auth.HashPassword(input.Password)
    if err != nil {
        return nil, fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
    }
    currentTime := time.Now()

    // 3. ビジネスロジック + 永続化 (関数として切り出す)。
    // createAccount の内側で BeginTx → 各 Repository の WithTx(tx) → Commit を行う。
    return uc.createAccount(ctx, input, passwordDigest, currentTime)
}
```

```go
// Mewst の例: 単一永続化で完結するため、ルール 2 の例外として Execute 直書き
type UpdatePasswordUsecase struct {
    passwordValidator *validator.PasswordUpdateValidator
    userRepo          *repository.UserRepository
}

type UpdatePasswordInput struct {
    Email    string
    Password string
}

// Execute はパスワード更新を実行する。
// バリデーション → ハッシュ化 → UPDATE の 1 ステップ書き込みで完結するため、
// オーケストレーションすべき対象がなく Execute 内で完結させている。
func (uc *UpdatePasswordUsecase) Execute(ctx context.Context, input UpdatePasswordInput) error {
    if err := uc.passwordValidator.Validate(ctx, validator.PasswordUpdateValidatorInput{
        Password: input.Password,
    }); err != nil {
        return err
    }

    passwordDigest, err := auth.HashPassword(input.Password)
    if err != nil {
        return fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
    }

    if err := uc.userRepo.UpdatePasswordByEmail(ctx, input.Email, passwordDigest); err != nil {
        return fmt.Errorf("パスワードの更新に失敗: %w", err)
    }
    return nil
}
```

### 永続化関数の粒度

ロジック + 永続化を切り出すプライベート関数は、原則として **1 つの UseCase につき 1 関数** にまとめる。複数リソースを作成する UseCase でも、`createXxx` のような単一の関数の中にトランザクション開始から Commit までを集約する。

リソースごとに細かく関数を切り出す (例: `createProfile`, `createUser`, `createUserProfile` 等) と、関数間でデータを受け渡すコストが増え、トランザクション境界が見えにくくなるため避ける。

### エラー型の使い分け

UseCase は以下の 3 種類のエラーを返す。Handler は `errors.As` でエラーの型を判別してレスポンスを決定する。

| エラー型                 | 生成元    | 意味                             | Handler の対応                          |
| ------------------------ | --------- | -------------------------------- | --------------------------------------- |
| `*model.ValidationError` | Validator | 入力が不正（ユーザーが修正可能） | フォーム再描画（422）                   |
| `*model.AppError`        | UseCase   | 業務レベルの既知の失敗           | エラーコードに応じた処理（403, 404 等） |
| 素の `error`             | どこでも  | 予期しないシステムエラー         | 500                                     |

各エラー型の詳細（フィールド構成、`AppError` の生成ルール、`AppErrorCode` 定数、ヘルパー関数）は [architecture-guide.md の「エラー型」節](architecture-guide.md#エラー型) を参照。

### Handler の実装パターン

Handler は薄い Adapter として、リクエストのパース → UseCase 呼び出し → レスポンス生成のみを行う。

```go
// Wikino の例
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. リクエストのパース
    title := r.FormValue("title")
    // ...

    // 2. UseCase 呼び出し（認可・バリデーション・永続化はすべて UseCase 内で実行）
    output, err := h.createSuggestionUC.Execute(ctx, usecase.CreateSuggestionInput{
        Title:           title,
        SpaceIdentifier: spaceIdentifier,
        UserID:          userID,
    })
    if err != nil {
        var ve *model.ValidationError
        if errors.As(err, &ve) {
            // バリデーションエラー → フォーム再描画（422）
            w.WriteHeader(http.StatusUnprocessableEntity)
            h.renderNewForm(w, r, ve)
            return
        }
        var ae *model.AppError
        if errors.As(err, &ae) {
            // アプリケーションエラー → ログ + エラーコードに応じた処理
            slog.ErrorContext(ctx, ae.LogString())
            h.renderError(w, r, ae)
            return
        }
        // 予期しないエラー → 500
        slog.ErrorContext(ctx, "予期しないエラー", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // 3. レスポンス
    http.Redirect(w, r, redirectPath, http.StatusSeeOther)
}
```

**Validator でのデータ取得パターン**: Validator は状態バリデーションの過程でデータを取得し、検証後にそのデータを戻り値として返す。これにより UseCase 内でデータを二重に取得する必要がなくなる。Validator は Go の慣習に従った `(data, error)` の 2 値返しを使用する。

```go
// Wikino の例
// Validator が検証の過程で取得したデータを戻り値として返す
func (v *SuggestionCreateValidator) Validate(ctx context.Context, input Input) ([]*model.DraftPage, error) {
    ve := model.NewValidationError()

    if input.Title == "" {
        ve.AddField("title", templates.T(ctx, "error_required"))
    }
    if ve.HasErrors() {
        return nil, ve  // *model.ValidationError は error を満たす
    }

    // 状態バリデーションで取得したデータを返す
    draftPages, err := v.draftPageRepo.FindByIDs(ctx, input.DraftPageIDs)
    if err != nil {
        return nil, err  // システムエラー
    }

    return draftPages, nil
}
```

## 実装例

### シンプルなユースケース（トランザクションなし）

```go
// internal/usecase/create_session.go
package usecase

import (
    "context"
    "example.com/app/internal/repository"
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

### 複雑なユースケース（トランザクションあり）

```go
// internal/usecase/create_password_reset_token.go
package usecase

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    "example.com/app/internal/repository"
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

## ハンドラーでの使用

Handler は薄い Adapter として UseCase を呼び出すだけです。具体的な実装パターンは「[UseCase 内の処理順序](#usecase-内の処理順序)」セクションの「Handler の実装パターン」を参照してください。

## Repository の WithTx パターン

Usecase でトランザクションを使用する場合、**Repository の `WithTx` メソッド**を使用してトランザクション内で操作するリポジトリを取得します。WithTx パターンの基本ルール (なぜ使うのか / Repository への実装 / Usecase での基本的な使い方 / 重要なポイント) は [architecture-guide.md](architecture-guide.md#repository-の-withtx-パターン) の「Repository の WithTx パターン」節を参照してください。

このセクションでは、書き込み UseCase の[ルール 1](#書き込み-usecase-のルール) (重い CPU 計算をトランザクション開始前に実行する) と組み合わせる発展的なパターンを示します。

### bcrypt と複数リソース作成を組み合わせる例

bcrypt のような重い CPU 計算と複数リソースの作成を組み合わせる場合は、書き込み UseCase のルール 1 (重い CPU 計算はトランザクション開始前に実行する) に従い、`Execute` 側で計算を済ませてからトランザクションを開く `createAccount` プライベート関数に値を渡します。

```go
// Mewst の例: Profile / User / UserProfile / Actor の 4 リソースを 1 トランザクションで作成する
package usecase

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "example.com/app/internal/auth"
    "example.com/app/internal/model"
    "example.com/app/internal/repository"
    "example.com/app/internal/validator"
)

const (
    ProfileOwnerTypeUser = "User"
    DefaultAvatarKind    = "default"
)

type CreateAccountUsecase struct {
    db               *sql.DB
    accountValidator *validator.AccountCreateValidator
    userRepo         *repository.UserRepository
    profileRepo      *repository.ProfileRepository
    userProfileRepo  *repository.UserProfileRepository
    actorRepo        *repository.ActorRepository
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

// Execute はバリデーション → bcrypt → createAccount のオーケストレーションのみを担当する。
// bcrypt と時刻取得は createAccount を呼ぶ前 (=トランザクション外) に済ませている。
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
    // 1. バリデーション (トランザクション外)
    if err := uc.accountValidator.Validate(ctx, validator.AccountCreateValidatorInput{
        Email:    input.Email,
        Atname:   input.Atname,
        Password: input.Password,
    }); err != nil {
        return nil, err
    }

    // 2. CPU 計算 (bcrypt) と時刻取得をトランザクション外で済ませる (ルール 1)。
    // bcrypt はコスト 10 で 100ms 級の処理になるため、トランザクション内で実行すると
    // その間 DB 接続を専有してロック競合の原因になる。
    passwordDigest, err := auth.HashPassword(input.Password)
    if err != nil {
        return nil, fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
    }
    currentTime := time.Now()

    // 3. ビジネスロジック + 永続化 (関数として切り出す)
    return uc.createAccount(ctx, input, passwordDigest, currentTime)
}

// createAccount は Profile / User / UserProfile / Actor を 1 トランザクションで作成する。
// 引数として受け取った passwordDigest / currentTime は Execute 側で計算済みのため、
// 本関数内ではトランザクション内で重い処理を行わない。
func (uc *CreateAccountUsecase) createAccount(
    ctx context.Context,
    input CreateAccountInput,
    passwordDigest string,
    currentTime time.Time,
) (*CreateAccountOutput, error) {
    tx, err := uc.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
    }
    defer func() { _ = tx.Rollback() }()

    userRepo := uc.userRepo.WithTx(tx)
    profileRepo := uc.profileRepo.WithTx(tx)
    userProfileRepo := uc.userProfileRepo.WithTx(tx)
    actorRepo := uc.actorRepo.WithTx(tx)

    profile, err := profileRepo.Create(ctx, repository.CreateProfileInput{
        OwnerType:  ProfileOwnerTypeUser,
        Atname:     input.Atname,
        JoinedAt:   currentTime,
        AvatarKind: DefaultAvatarKind,
    })
    if err != nil {
        return nil, fmt.Errorf("プロフィールの作成に失敗: %w", err)
    }

    user, err := userRepo.Create(ctx, repository.CreateUserInput{
        Email:          input.Email,
        PasswordDigest: passwordDigest,
        Locale:         input.Locale,
        TimeZone:       input.TimeZone,
    })
    if err != nil {
        return nil, fmt.Errorf("ユーザーの作成に失敗: %w", err)
    }

    if _, err := userProfileRepo.Create(ctx, repository.CreateUserProfileInput{
        UserID:    user.ID,
        ProfileID: profile.ID,
    }); err != nil {
        return nil, fmt.Errorf("ユーザープロフィール関連付けの作成に失敗: %w", err)
    }

    actor, err := actorRepo.Create(ctx, repository.CreateActorInput{
        UserID:    user.ID,
        ProfileID: profile.ID,
    })
    if err != nil {
        return nil, fmt.Errorf("アクターの作成に失敗: %w", err)
    }

    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
    }

    return &CreateAccountOutput{Actor: actor}, nil
}
```

## テスト

```go
func TestCreatePasswordResetTokenUsecase_Execute(t *testing.T) {
    // 共有DB接続プールからトランザクションをセットアップ
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

## 命名の注意点

### ファイル名の順序

`{action}_{entity}` の順（動詞を必ず先頭に）

- ✅ `create_session.go`
- ❌ `session_create.go`
- ✅ `create_password_reset_token.go`
- ❌ `password_reset_create_token.go`

### 複合エンティティ

エンティティが複数単語の場合はアンダースコアで連結

- ✅ `create_password_reset_token.go` （password_reset_token というエンティティ）

### 構造体名の大文字化

`Usecase` の `c` は小文字

- ✅ `CreateSessionUsecase`
- ❌ `CreateSessionUseCase`

## ファイル配置の理由

### フラット構造

エンティティごとにディレクトリを作らず、`internal/usecase/` 直下にすべてのファイルを配置

理由:

- **検索性**: ファイル名のプレフィックスでグルーピングされるため、エディタで検索しやすい
- **シンプルさ**: ディレクトリ階層が深くならず、import パスがシンプル
- **スケーラビリティ**: ファイル数が増えても管理しやすい

## 採用しなかった方針

### A. 書き込み UseCase のために読み取り UseCase を新設する

書き込み UseCase からすべてのデータ取得を外出しし、書き込み UseCase のためだけに読み取り UseCase を作成する方針。

**不採用の理由**:

- Handler が書き込み UseCase の内部実装を知る必要が生じる（どんなデータを事前に用意すべきか）
- 書き込み UseCase のために読み取り UseCase を作ると、両者が強く結合し、分離のメリットが薄い
- 命名が酷似し混同しやすくなる（例: `GetDraftPageSaveDataUsecase` と `GetSaveDraftPageDataUsecase`）

**代替として採用した方針**: 書き込み UseCase 内であっても、トランザクション開始前であればデータ取得を行ってよい。書き込み UseCase のルール（トランザクション内は永続化のみ、Execute 内にロジックを直接書かない）を守る限り、データ取得の配置場所は柔軟に判断する。

### B. Handler がオーケストレーターとして認可・バリデーションを制御する

Handler が読み取り UseCase → Policy → Validator → 書き込み UseCase の流れを制御する方針。

**不採用の理由**:

- エントリーポイントが増えた場合（Web API など）、認可・バリデーションの呼び出しを各エントリーポイントで再現する必要があり、漏れが発生しやすい
- 外部世界との接点である Handler にビジネスロジックの制御フローが書かれており、関心の分離が不十分
- Handler にドメイン固有の判断が集中し、テストが複雑になる

**代替として採用した方針**: UseCase をオーケストレーターにする。バリデーション・認可・ビジネスロジック・永続化を UseCase 内部で統括し、Handler は HTTP の入出力変換に徹する。

### C. Read UseCase を廃止し UseCase を1つに統合する

GET と POST で同じ UseCase を呼び、引数で動作を切り替える方針。

**不採用の理由**:

- GET（フォーム表示）と POST（作成処理）で責務が異なるため、1つの UseCase に統合すると不自然になる
- 読み取り UseCase はフォーム表示専用として残すほうが、責務が明確でシンプル

### D. 書き込み UseCase とオーケストレーション UseCase を一本化する

Validator を必須にした書き込み UseCase に統合し、3 分類を「読み取り / 書き込み (Validator 必須)」の 2 分類にする方針。「書き込み UseCase は必ず Validator を持つ」というルールが分類にエンコードされ、概念がシンプルになる。

**不採用の理由**:

- **Validator を持たない書き込み UseCase が現実に必要**: セッション作成 (例: `CreateSessionUsecase`) のように、入力が「別 UseCase の戻り値 + HTTP コンテキスト由来の確定済みデータ (UserID / IPAddress / UserAgent)」だけで、ユーザー入力のバリデーションが不要な書き込み UseCase が存在する。バックグラウンドジョブから呼ばれる UseCase (例: メール送信) も、入力が別 UseCase で確定済みのため Validator は不要。これらに対してダミー Validator を作るのは過剰
- **Handler の前処理は UseCase に押し込めない**: Bot 対策トークン検証 (例: Cloudflare Turnstile)・IP アドレスベースのレート制限・クッキーからの ID 取得などは「HTTP リクエスト固有の制御」であり、ユーザー入力のバリデーションとは性質が異なる (フォームに依存せず複数機能で再利用、HTTP コンテキスト依存)。これらまで UseCase に入れると、UseCase が HTTP 文脈を背負い、Worker から再利用しにくくなる
- **判断軸が分類にエンコードされる利点を失う**: 3 分類は「Validator の有無」という実装判断を分類名にエンコードしているため、「フォームの入口なら必ずオーケストレーション」「内部呼び出しの純粋永続化なら書き込み」と判断軸が明確。一本化すると「この UseCase は Validator を持つべきか?」を毎回判断する必要が生じる

**将来検討する余地**: 「内部から呼ばれる純粋永続化 UseCase」を別パッケージ (例: `internal/service/`) に分離すれば、`internal/usecase/` に残るのはすべて Validator を統合する書き込み UseCase に一本化できる可能性がある。ただしこれは現行コードベース全体に影響する再設計であり、内部呼び出し UseCase の数が増えて分離のメリットが見えてきた段階で改めて検討する
