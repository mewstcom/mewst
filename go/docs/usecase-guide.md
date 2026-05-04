# UseCase ガイド

このドキュメントは、Go 版 Mewst の UseCase 層の設計と実装パターンを説明します。

## 概要

UseCase はアプリケーションのオーケストレーターです。Handler / Worker からのすべてのデータアクセス・認可・バリデーション・ビジネスロジック・永続化は UseCase を経由します。

## 責務

- **オーケストレーション**: 認可チェック・バリデーション・ビジネスロジック・永続化を統括する
- データ取得ロジックの集約 (読み取り UseCase)
- トランザクション管理 (書き込み UseCase: `db.BeginTx` から `tx.Commit` まで)
- 複数の Repository を跨ぐ処理

## UseCase の分類

Mewst では UseCase を以下の 3 種類に分類します。「オーケストレーション UseCase」は書き込み UseCase の特殊形 (Validator を統合してフォーム送信全体を統括する書き込み UseCase) ですが、Mewst では Validator を持つかどうかで実装パターンが大きく変わるため独立カテゴリとして扱います。

| 種類                         | 責務                                                            | Validator | トランザクション                |
| ---------------------------- | --------------------------------------------------------------- | --------- | ------------------------------- |
| 読み取り UseCase             | データ取得、複数 Repository の集約                              | なし      | なし                            |
| 書き込み UseCase             | 永続化処理 (作成・更新・削除)、ビジネスロジック                 | なし      | あり/なし (必要に応じて WithTx) |
| オーケストレーション UseCase | Validator を統合し、フォーム送信のバリデーション → 永続化を統括 | あり      | あり/なし (必要に応じて WithTx) |

書き込み UseCase / オーケストレーション UseCase は、複数の永続化を伴う場合のみトランザクションを開く。単一の永続化呼び出しで完結する場合はトランザクションを伴わない (例: `CreateSessionUsecase` は書き込み UseCase だがトランザクションなし、`UpdatePasswordUsecase` はオーケストレーション UseCase だがトランザクションなし)。

各分類の役割:

**読み取り UseCase**

- Handler が必要とするデータ取得ロジックを集約する
- 複数の Repository を組み合わせてデータを取得する
- トランザクションは不要

**書き込み UseCase**

- 永続化処理 (作成・更新・削除) とビジネスロジック
- 複数の Repository を跨ぐ場合や、ロールバックが必要な複合操作ではトランザクションを開く
- 単一の永続化呼び出しで完結する場合はトランザクションを伴わない (例: `CreateSessionUsecase`)
- 後述の「書き込み UseCase の 2 つのルール」を守る

**オーケストレーション UseCase**

- 書き込み UseCase の中でも、Validator を統合してフォーム送信全体を統括するもの
- Handler は HTTP の入出力変換に徹し、認可・バリデーション・ビジネスロジック・永続化は UseCase 内部で完結する
- Mewst の現存する書き込み UseCase の多く (`CreateAccount` / `CreateSignIn` / `CreateSignUp` / `CreatePasswordReset` / `VerifyEmailConfirmation` / `UpdatePassword`) はこのカテゴリに入る

### どの分類にすべきかの判断フロー

新しい UseCase を追加するときは、以下の順で判断します。

```
1. Repository の取得系メソッド (Find*) のみを呼ぶ?
   → 読み取り UseCase (プレフィックス: Get)

2. Validator を統合する? (フォーム送信の入口を担う)
   → オーケストレーション UseCase

3. 上記以外 (永続化を伴うが Validator なし)
   → 書き込み UseCase
   例: Validator を持たない `CreateSessionUsecase` (Handler 側で先にバリデーションが終わっている前提でセッションを作成するだけの UseCase)
```

「永続化を伴うが Validator を持たない」ケースは比較的少数 (Mewst では `CreateSessionUsecase` のみ) ですが、Handler ではなく内部の別 UseCase から呼ばれる書き込み UseCase はこの形になります。

## ファイル配置

`internal/usecase/` 直下にフラットに配置 (サブディレクトリは作成しない)。

**プライベート関数の配置ルール**: ある UseCase ファイルに定義されたプライベート関数を別の UseCase ファイルから呼び出す必要が生じた場合は、その関数を専用のファイルに切り出す。ファイル名は関数の責務を表す名詞にする。

例: 確認コード生成 (`generateConfirmationCode`) は `CreateSignUpUsecase` と `CreatePasswordResetUsecase` の両方から呼ばれるため、`internal/usecase/confirmation_code.go` に切り出されている。

## 命名規則

- **ファイル名**: `{action}_{entity}.go`
  - 例: `create_session.go`, `create_password_reset.go`, `update_password.go`
  - **重要**: 動詞 (アクション) を必ず先頭に配置する
- **構造体名**: `{Action}{Entity}Usecase`
  - 例: `CreateSessionUsecase`, `CreatePasswordResetUsecase`
  - 注: `Usecase` の `c` は小文字 (既存コードとの統一のため)
- **読み取り UseCase のプレフィックスは `Get` に統一**: 読み取り UseCase のアクションには `Get` を使用する。`List` や `Fetch` など他の動詞は使用しない。これにより `Get` = 読み取り、それ以外 = 書き込みという判別が即座にできる
  - 例: `GetActiveEmailConfirmationUsecase`, `GetSucceededEmailConfirmationUsecase`
- **コンストラクタ**: `New{Action}{Entity}Usecase`
- **Execute メソッド**: `Execute(ctx context.Context, input ...) (*Output, error)` (戻り値が単一の場合は `(*Output, error)`、出力データが不要な場合は `error` のみでも可)

### ファイル名の順序

`{action}_{entity}` の順 (動詞を必ず先頭に)。

- ✅ `create_session.go`
- ❌ `session_create.go`
- ✅ `create_password_reset.go`
- ❌ `password_reset_create.go`

### 複合エンティティ

エンティティが複数単語の場合はアンダースコアで連結する。

- ✅ `create_password_reset_token.go` (password_reset_token というエンティティ)

### 構造体名の大文字化

`Usecase` の `c` は小文字。

- ✅ `CreateSessionUsecase`
- ❌ `CreateSessionUseCase`

## Input / Output 構造体

各 UseCase は専用の Input / Output 構造体を持ちます。引数が増えてもシグネチャが安定し、呼び出し側の差分が局所化できます。

- **Input**: `{UseCase 名}Input` (`CreateAccountInput`, `GetActiveEmailConfirmationInput` 等)
- **Output**: `{UseCase 名}Output` (`CreateAccountOutput`, `CreateSessionOutput` 等)。Mewst では `Output` suffix で統一する

```go
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
```

## UseCase 内の処理順序

書き込み UseCase / オーケストレーション UseCase は以下の順序で処理を実行します。すべてのステップが必須ではなく、UseCase によっては一部ステップが省略されます (例: 自由公開な UseCase では認可ステップが省略される)。

```
1. データ取得 (トランザクション外)
2. 認可チェック
3. バリデーション (Validator)
4. ビジネスロジック (計算・変換等)
5. トランザクション (永続化のみ)
```

**注意**: Mewst では `SignInCreateValidator` のように Validator が状態バリデーションでデータ取得を行うパターン ([Validator のデータ取得パターン](#validator-でのデータ取得パターン)) を採用しており、その場合はステップ 1 (データ取得) が Validator に統合される。Validator が返したデータは Execute 側で再取得せず、そのまま再利用する (二度引きしない)。Execute 側で追加のデータ取得 (例: Validator が引いた User の ID から Actor を取得する) が必要な場合は、バリデーション後・トランザクション開始前に行う。

### 認可チェックの位置づけ (Mewst 固有)

Mewst には現状 Policy パッケージがなく、認可は以下のいずれかで実装しています:

- **認証**: `internal/middleware/auth.go` のミドルウェアでセッション Cookie を検証し、未ログインのリクエストはハンドラーに到達する前に弾く
- **リソース所有者チェック等の認可**: Validator の状態バリデーションで実装する (例: 「このメール確認は本当にこのユーザーのものか?」を Validator 内で検証する)

将来的に「ページ閲覧権限」「メンバー権限による分岐」など認可ロジックが集中した場合は、Wikino と同様に `internal/policy/` パッケージとして切り出す可能性があります。その時点で本ガイドの 5 ステップ表記の「認可チェック」を Policy に置き換えます。

### 書き込み UseCase の 2 つのルール

書き込み UseCase は以下の 2 つのルールを守ります。

1. **トランザクション開始後はデータの取得や計算処理を行わない**: トランザクション内は永続化処理のみ行う。データ取得は `BeginTx` の前で済ませ、`auth.HashPassword` (bcrypt) のような重い CPU 計算もトランザクション開始前に実行する。トランザクションを長時間保持しないことで、ロックや競合の発生を抑える
2. **`Execute` をオーケストレーションに専念させる**: 永続化を含む実処理はプライベート関数 (またはメソッド) として定義し、`Execute` 内ではそれを呼び出すだけにする。`Execute` は「データ取得・認可・バリデーション・永続化関数の呼び出し」のオーケストレーションのみに専念する。

   **例外**: ルール 2 の主旨は「`Execute` をオーケストレーションに専念させる」ことであり、オーケストレーションすべき対象がない場合は本ルールの適用範囲外となる。具体的には、単一の永続化呼び出し + 必要最小限の前処理だけで完結する書き込み UseCase は、プライベート関数化を省略して `Execute` に直接書いてよい。例: `UpdatePasswordUsecase` は「バリデーション → ハッシュ化 → UPDATE」だけで他にオーケストレーションするものがないため、`Execute` 内で完結させる。

   **関数切り出しが必要かどうかの判断フロー**:

   ```
   1. トランザクション内で永続化処理が 2 つ以上ある?
      → 必要 (プライベート関数化)
      例: CreateAccountUsecase (Profile / User / UserProfile / Actor の 4 リソースを作成)

   2. 永続化前にロジック (計算・変換) が 2 行以上ある?
      → 必要 (プライベート関数化)
      例: 確認コード生成 + 永続化を行う CreateSignUpUsecase / CreatePasswordResetUsecase

   3. 上記以外 (バリデーション → 1 つの永続化、または最小限の変換 → 1 つの永続化)
      → 不要 (Execute 直書きで OK)
      例: UpdatePasswordUsecase (バリデーション → ハッシュ化 → 1 回の UPDATE)
   ```

   閾値 (「2 つ以上」「2 行以上」) は機械的に守るべき絶対値ではなく、「`Execute` がオーケストレーターとしての役割を持つかどうか」を見極めるための目安として扱う。境界例ではプロジェクトに既存する近い UseCase の書き方に合わせる。

```go
// ✅ 良い例: 書き込み UseCase がデータ取得・バリデーション・永続化を統括する
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
    // 1. バリデーション (トランザクション外)
    if err := uc.accountsValidator.Validate(ctx, validator.AccountsCreateValidatorInput{
        Email:    input.Email,
        Atname:   input.Atname,
        Password: input.Password,
    }); err != nil {
        return nil, err // *model.ValidationError か素の error がそのまま上がる
    }

    // 2. CPU 計算 (bcrypt) と時刻取得をトランザクション外で済ませる (ルール 1)
    passwordDigest, err := auth.HashPassword(input.Password)
    if err != nil {
        return nil, fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
    }
    currentTime := time.Now()

    // 3. ビジネスロジック + 永続化 (関数として切り出す)
    return uc.createAccount(ctx, input, passwordDigest, currentTime)
}

// ❌ 悪い例: トランザクション内でデータ取得や CPU 計算を行っている
func (uc *WriteUsecase) Execute(ctx context.Context, input Input) error {
    tx, err := uc.db.BeginTx(ctx, nil)
    // トランザクション内でデータ取得 → トランザクション前に行うべき
    user, err := userRepo.FindByID(ctx, input.UserID)
    // ...
    // bcrypt のハッシュ化もトランザクション内で実行 → トランザクション前に行うべき
    digest, err := auth.HashPassword(input.Password)
    // ...
}
```

### 永続化関数の粒度

ロジック + 永続化を切り出すプライベート関数は、原則として **1 つの UseCase につき 1 関数** にまとめます。CreateAccount のように複数リソースを作成する場合でも、`createAccount` という単一の関数の中にトランザクション開始から Commit までを集約します。

リソースごとに細かく関数 (`createProfile`, `createUser`, `createUserProfile` 等) を切り出すと、関数間でデータを受け渡すコストが増え、トランザクション境界が見えにくくなるため避けます。

## エラー型の使い分け

UseCase は以下の 3 種類のエラーを返します。Handler は `errors.As` でエラーの型を判別してレスポンスを決定します。

| エラー型                 | 生成元    | 意味                            | Handler の対応                         |
| ------------------------ | --------- | ------------------------------- | -------------------------------------- |
| `*model.ValidationError` | Validator | 入力が不正 (ユーザーが修正可能) | フォーム再描画 (422)                   |
| `*model.AppError`        | UseCase   | 業務レベルの既知の失敗          | エラーコードに応じた処理 (403, 404 等) |
| 素の `error`             | どこでも  | 予期しないシステムエラー        | 500                                    |

エラーラップは `fmt.Errorf("...: %w", err)` を使用し、内部エラーをラップしながら呼び出し元へ伝搬させます。

```go
ec, err := uc.emailConfirmationRepo.FindActiveByID(ctx, input.ID)
if err != nil {
    return nil, fmt.Errorf("有効なメール確認の取得に失敗: %w", err)
}
if ec == nil {
    return nil, ErrNotFound
}
```

「未存在」を業務レベルの異常として扱う場合は、`internal/usecase/errors.go` で定義された `usecase.ErrNotFound` を返すか、`*model.AppError` (`AppErrCodeResourceNotFound`) を返します。Handler 側はそれぞれ 404 や該当ページにリダイレクトといった対応を行います。

エラー型の詳細 (`ValidationError` / `AppError` の構造、`AsValidationError` / `AsAppError` ヘルパー) はアーキテクチャガイドの「[エラー型](architecture-guide.md#エラー型)」節を参照してください。

## Validator でのデータ取得パターン

Validator は状態バリデーションの過程でデータを取得し、検証後にそのデータを戻り値として返します。これにより UseCase 側でデータを二重に取得する必要がなくなります。Validator は Go の慣習に従った `(data, error)` の 2 値返しを使用します。出力が単一モデルだけで済む場合は Output 構造体を作らずモデルを直接返します ([validation-guide.md](validation-guide.md#構造体の命名規則) 参照)。

例として `SignInCreateValidator` を取り上げます。Validator はサインイン入力の形式チェックを行ったあと、状態バリデーションでメールアドレスから User を引いてパスワード照合まで済ませ、検証成功時には `*model.User` を直接返します。`CreateSignInUsecase` はそれを再取得せず、User の ID で Actor を引いてセッション作成まで進めます。

```go
// internal/validator/sign_in.go (Mewst の現行実装、抜粋)
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*model.User, error) {
    ve := model.NewValidationError()

    // 形式バリデーション (メール形式 / パスワード必須) は省略

    // 状態バリデーション: DB から User を引いてパスワード照合まで済ませる
    user, err := v.userRepo.FindByEmail(ctx, input.Email)
    if err != nil {
        return nil, err
    }
    if user == nil {
        ve.AddGlobal(i18n.T(ctx, "error_invalid_credentials"))
        return nil, ve
    }
    if err := auth.CheckPassword(user.PasswordDigest, input.Password); err != nil {
        ve.AddGlobal(i18n.T(ctx, "error_invalid_credentials"))
        return nil, ve
    }

    return user, nil
}
```

```go
// internal/usecase/create_sign_in.go (UseCase 側で再利用)
func (uc *CreateSignInUsecase) Execute(ctx context.Context, input CreateSignInInput) (*CreateSignInOutput, error) {
    // 1. バリデーション (User を引いてパスワード照合まで済ませる)
    user, err := uc.signInValidator.Validate(ctx, validator.SignInCreateValidatorInput{
        Email:    input.Email,
        Password: input.Password,
    })
    if err != nil {
        return nil, err
    }

    // 2. Validator が引いた User の ID を使って Actor を取得 (二度引きしない)
    actor, err := uc.actorRepo.FindByUserID(ctx, user.ID)
    if err != nil {
        return nil, fmt.Errorf("アクターの取得に失敗: %w", err)
    }
    if actor == nil {
        return nil, ErrNotFound
    }

    // 3. ビジネスロジック + 永続化 (関数として切り出す)
    return uc.createSignIn(ctx, actor.ID, input)
}
```

このように、Validator が状態バリデーションで取得したエンティティ (ここでは `User`) をそのまま受け取って UseCase 側で使うことで、`userRepo.FindByEmail` を二度呼ばずに済みます。Validator が引いたデータをそのまま永続化に渡すか、さらに別エンティティ (Actor 等) の取得に使うかは UseCase 側の判断で構いません。「二度引きしない」ことが目的です。

## Repository の WithTx パターン

書き込み UseCase でトランザクションを使用する場合、Repository の `WithTx` メソッドを使ってトランザクション内で操作するリポジトリを取得します。

### 目的

- Repository をコンストラクタで受け取り、`Execute` (またはプライベート関数) 内で `WithTx(tx)` を呼び出すことで、トランザクション内で安全にデータ操作を行う
- 同じ Repository を、通常時 (DB 直接) とトランザクション時 (tx 経由) の両方で使用できる

### 重要なポイント

1. **Repository はコンストラクタで受け取る**: `NewXxxUsecase` で Repository を引数として受け取る
2. **永続化関数の入口で `WithTx` を呼び出す**: トランザクションを開始した直後、各 Repository の `WithTx(tx)` を呼び出す
3. **`defer func() { _ = tx.Rollback() }()`**: `Commit` 成功後の `Rollback` は no-op なので、常に defer で呼び出して安全にロールバックを保証する
4. **元の Repository は変更されない**: `WithTx` は新しい Repository インスタンスを返すため、元の Repository には影響しない
5. **すべての Repository で `WithTx` を使う**: トランザクション内で使用するすべての Repository に対して `WithTx` を呼び出す

## 実装例

### 読み取り UseCase

トランザクションなし、データ取得のみ。`(nil, nil)` パターンの Repository から nil が返ってきた場合は `usecase.ErrNotFound` に変換して上位層へ伝搬します。

```go
// internal/usecase/get_active_email_confirmation.go
package usecase

import (
    "context"
    "fmt"

    "github.com/mewstcom/mewst/go/internal/model"
    "github.com/mewstcom/mewst/go/internal/repository"
)

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
    ID model.EmailConfirmationID
}

type GetActiveEmailConfirmationOutput struct {
    EmailConfirmation *model.EmailConfirmation
}

func (uc *GetActiveEmailConfirmationUsecase) Execute(ctx context.Context, input GetActiveEmailConfirmationInput) (*GetActiveEmailConfirmationOutput, error) {
    ec, err := uc.emailConfirmationRepo.FindActiveByID(ctx, input.ID)
    if err != nil {
        return nil, fmt.Errorf("有効なメール確認の取得に失敗: %w", err)
    }
    if ec == nil {
        return nil, ErrNotFound
    }
    return &GetActiveEmailConfirmationOutput{EmailConfirmation: ec}, nil
}
```

### オーケストレーション UseCase (複数リソース + トランザクション)

`CreateAccountUsecase` は Mewst で最大の書き込み UseCase で、Profile / User / UserProfile / Actor の 4 リソースを 1 トランザクションで作成します。Validator を統合してフォーム送信全体を統括するため、オーケストレーション UseCase に分類されます。

`Execute` は「バリデーション → `createAccount` 呼び出し」のオーケストレーションのみで、トランザクション開始から Commit までは `createAccount` プライベート関数に集約しています (書き込み UseCase のルール 2)。

```go
// internal/usecase/create_account.go
package usecase

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "github.com/mewstcom/mewst/go/internal/auth"
    "github.com/mewstcom/mewst/go/internal/model"
    "github.com/mewstcom/mewst/go/internal/repository"
    "github.com/mewstcom/mewst/go/internal/validator"
)

// ProfileOwnerTypeUser はプロフィールの所有者タイプ（ユーザー）
const ProfileOwnerTypeUser = "User"

// DefaultAvatarKind はデフォルトのアバター種別
const DefaultAvatarKind = "default"

type CreateAccountUsecase struct {
    db                *sql.DB
    accountsValidator *validator.AccountsCreateValidator
    userRepo          *repository.UserRepository
    profileRepo       *repository.ProfileRepository
    userProfileRepo   *repository.UserProfileRepository
    actorRepo         *repository.ActorRepository
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

// Execute はアカウントを作成する
// Profile, User, UserProfile, Actor を一括で作成し、トランザクション管理を行う
func (uc *CreateAccountUsecase) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
    // 1. バリデーション (トランザクション外)
    if err := uc.accountsValidator.Validate(ctx, validator.AccountsCreateValidatorInput{
        Email:    input.Email,
        Atname:   input.Atname,
        Password: input.Password,
    }); err != nil {
        return nil, err
    }

    // 2. CPU 計算 (bcrypt) と時刻取得をトランザクション外で済ませる。
    // bcrypt はコスト 10 で 100ms 級の処理になるため、トランザクション内で実行すると
    // その間 DB 接続を専有してロック競合の原因になる。
    passwordDigest, err := auth.HashPassword(input.Password)
    if err != nil {
        return nil, fmt.Errorf("パスワードのハッシュ化に失敗: %w", err)
    }
    currentTime := time.Now()

    // 3. ビジネスロジック + 永続化
    return uc.createAccount(ctx, input, passwordDigest, currentTime)
}

// createAccount は Profile / User / UserProfile / Actor を 1 トランザクションで作成する
func (uc *CreateAccountUsecase) createAccount(ctx context.Context, input CreateAccountInput, passwordDigest string, currentTime time.Time) (*CreateAccountOutput, error) {
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
        OwnerType:     ProfileOwnerTypeUser,
        Atname:        input.Atname,
        Name:          "",
        Description:   "",
        ImageURL:      "",
        JoinedAt:      currentTime,
        AvatarKind:    DefaultAvatarKind,
        GravatarEmail: "",
        GravatarURL:   "",
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

    return &CreateAccountOutput{
        Actor: actor,
    }, nil
}
```

### オーケストレーション UseCase (単一永続化、トランザクションなし)

`UpdatePasswordUsecase` のように Validator を統合しつつも単一の永続化呼び出しで完結する UseCase は、トランザクションを開かず `Execute` に直接処理を書きます。書き込み UseCase の[ルール 2 の例外](#書き込み-usecase-の-2-つのルール) (関数切り出しが必要かどうかの判断フローのケース 3、永続化 1 回 + 前処理 2 行未満) に該当します。Validator を統合するためオーケストレーション UseCase に分類されますが、トランザクションを開かないため `WithTx` は使用しません。

```go
// internal/usecase/update_password.go
package usecase

import (
    "context"
    "fmt"

    "github.com/mewstcom/mewst/go/internal/auth"
    "github.com/mewstcom/mewst/go/internal/repository"
    "github.com/mewstcom/mewst/go/internal/validator"
)

type UpdatePasswordUsecase struct {
    passwordValidator *validator.PasswordUpdateValidator
    userRepo          *repository.UserRepository
}

func NewUpdatePasswordUsecase(
    passwordValidator *validator.PasswordUpdateValidator,
    userRepo *repository.UserRepository,
) *UpdatePasswordUsecase {
    return &UpdatePasswordUsecase{
        passwordValidator: passwordValidator,
        userRepo:          userRepo,
    }
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

`Execute` 直書きで完結するため、`Output` 構造体も不要なケースでは戻り値を `error` のみにできます。Validator 側もデータを返さない場合は `error` のみのシグネチャにし (例: `PasswordUpdateValidator.Validate(ctx, input) error`)、UseCase 側では `if err := uc.passwordValidator.Validate(...)` の形で受けます。

### 書き込み UseCase (トランザクションなしで完結する書き込み)

トランザクションを跨ぐ複数 Repository 操作がない単一の永続化処理は、トランザクションを開かずに直接 Repository を呼び出します。CreateSession のように「actor を引いてセッションを作成する」だけのケースが該当します。

```go
// internal/usecase/create_session.go
package usecase

import (
    "context"
    "fmt"

    "github.com/mewstcom/mewst/go/internal/auth"
    "github.com/mewstcom/mewst/go/internal/model"
    "github.com/mewstcom/mewst/go/internal/repository"
)

type CreateSessionUsecase struct {
    actorRepo   *repository.ActorRepository
    sessionRepo *repository.SessionRepository
}

func NewCreateSessionUsecase(actorRepo *repository.ActorRepository, sessionRepo *repository.SessionRepository) *CreateSessionUsecase {
    return &CreateSessionUsecase{
        actorRepo:   actorRepo,
        sessionRepo: sessionRepo,
    }
}

type CreateSessionInput struct {
    UserID    model.UserID
    IPAddress string
    UserAgent string
}

type CreateSessionOutput struct {
    Session *model.Session
    Token   string
}

// Execute はセッションを作成する
func (uc *CreateSessionUsecase) Execute(ctx context.Context, input CreateSessionInput) (*CreateSessionOutput, error) {
    // 1. データ取得 (トランザクション外)
    actor, err := uc.actorRepo.FindByUserID(ctx, input.UserID)
    if err != nil {
        return nil, fmt.Errorf("アクターの取得に失敗: %w", err)
    }
    if actor == nil {
        return nil, ErrNotFound
    }

    // 2. ビジネスロジック + 永続化
    return uc.createSession(ctx, actor.ID, input)
}

func (uc *CreateSessionUsecase) createSession(ctx context.Context, actorID model.ActorID, input CreateSessionInput) (*CreateSessionOutput, error) {
    token, err := auth.GenerateSecureToken()
    if err != nil {
        return nil, fmt.Errorf("セッショントークンの生成に失敗: %w", err)
    }

    s, err := uc.sessionRepo.Create(ctx, repository.CreateSessionInput{
        ActorID:   actorID,
        Token:     token,
        IPAddress: input.IPAddress,
        UserAgent: input.UserAgent,
    })
    if err != nil {
        return nil, fmt.Errorf("セッションの作成に失敗: %w", err)
    }

    return &CreateSessionOutput{
        Session: s,
        Token:   token,
    }, nil
}
```

## Handler での使用

Handler は薄い Adapter として、リクエストのパース → UseCase 呼び出し → レスポンス生成のみを行います。`errors.As` で `*model.ValidationError` / `*model.AppError` を判別し、それぞれに対応する HTTP ステータスを返します。

具体的なコード例は [handler-guide.md](handler-guide.md) の「実装例」と「エラーハンドリング」を参照してください。要点は以下の通りです:

- データアクセスは UseCase 経由のみ (Handler から `repository` / `query` / `validator` への直接依存は depguard で禁止)
- エラー判別は `errors.As`: `*model.ValidationError` ならフォーム再描画 (422)、`*model.AppError` ならエラーコードに応じた HTTP ステータス、それ以外は 500
- 副作用 (Cookie 書き込み・Flash 設定・リダイレクト) は UseCase 成功後に Handler が行う

## テスト

UseCase のテストは `testutil.SetupTx(t)` で共有 DB 接続プールからトランザクションを取得し、テスト終了時に自動ロールバックする方式で書きます。Validator もテスト用トランザクションで初期化することで、状態バリデーション込みの統合テストが可能です。

### 読み取り UseCase のテスト例

```go
func TestGetActiveEmailConfirmationUsecase_Execute_Success(t *testing.T) {
    t.Parallel()

    // テスト DB とトランザクションをセットアップ (テスト終了時に自動ロールバック)
    _, tx := testutil.SetupTx(t)
    ctx := context.Background()

    // ビルダーでテストデータを作成
    emailConfirmationID := testutil.NewEmailConfirmationBuilder(t, tx).
        WithEmail("test@example.com").
        WithEvent("password_reset").
        WithCode("123456").
        Build()

    // Repository をテスト用トランザクションで初期化
    emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))

    // UseCase を実行
    uc := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmRepo)
    result, err := uc.Execute(ctx, usecase.GetActiveEmailConfirmationInput{
        ID: emailConfirmationID,
    })

    // アサーション
    if err != nil {
        t.Fatalf("Execute() error = %v", err)
    }
    if result.EmailConfirmation.Email != "test@example.com" {
        t.Errorf("Email = %v, want test@example.com", result.EmailConfirmation.Email)
    }
}

func TestGetActiveEmailConfirmationUsecase_Execute_NotFound(t *testing.T) {
    t.Parallel()

    _, tx := testutil.SetupTx(t)
    ctx := context.Background()

    emailConfirmRepo := repository.NewEmailConfirmationRepository(testutil.QueriesWithTx(tx))
    uc := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmRepo)

    // 存在しない ID で実行
    _, err := uc.Execute(ctx, usecase.GetActiveEmailConfirmationInput{
        ID: model.EmailConfirmationID(uuid.New()),
    })

    // 未存在は usecase.ErrNotFound として伝搬される
    if !errors.Is(err, usecase.ErrNotFound) {
        t.Errorf("Execute() error should be ErrNotFound, got %v", err)
    }
}
```

### オーケストレーション UseCase のテスト方針

オーケストレーション UseCase のテストでは、Validator も含めて統合的に検証します。`*model.ValidationError` の戻り値を `model.AsValidationError` で判別し、フィールドエラーを検証することでバリデーション統合の動作を保証します。

詳細なテストパターンは将来作成される `testing-guide.md` (フェーズ 11-2 で新設予定) を参照してください。

## 採用しなかった方針

### A. 書き込み UseCase のために読み取り UseCase を新設する

書き込み UseCase からすべてのデータ取得を外出しし、書き込み UseCase のためだけに読み取り UseCase を作成する方針。

**不採用の理由**:

- Handler が書き込み UseCase の内部実装を知る必要が生じる (どんなデータを事前に用意すべきか)
- 書き込み UseCase のために読み取り UseCase を作ると、両者が強く結合し、分離のメリットが薄い
- 命名が酷似し混同しやすくなる

**代替として採用した方針**: 書き込み UseCase 内であっても、トランザクション開始前であればデータ取得を行ってよい。書き込み UseCase の 2 つのルール (トランザクション内は永続化のみ、`Execute` をオーケストレーションに専念させる) を守る限り、データ取得の配置場所は柔軟に判断する。

なお、Mewst の Repository は「未存在」時に `(nil, nil)` を返すパターンを採用しているため、UseCase 側では `if data == nil` で未存在を判定し、必要に応じて `usecase.ErrNotFound` に変換して上位層に伝搬する。

### B. Handler がオーケストレーターとして認可・バリデーションを制御する

Handler が読み取り UseCase → Validator → 書き込み UseCase の流れを制御する方針。

**不採用の理由**:

- エントリーポイントが増えた場合 (Web API、Worker など)、認可・バリデーションの呼び出しを各エントリーポイントで再現する必要があり、漏れが発生しやすい
- 外部世界との接点である Handler にビジネスロジックの制御フローが書かれており、関心の分離が不十分
- Handler にドメイン固有の判断が集中し、テストが複雑になる

**代替として採用した方針**: UseCase をオーケストレーターにする。バリデーション・認可・ビジネスロジック・永続化を UseCase 内部で統括し、Handler は HTTP の入出力変換に徹する。Worker からも同じ UseCase を呼び出せるため、複数のエントリーポイントから一貫した処理を保証できる。

## 関連ドキュメント

- [@go/CLAUDE.md](../CLAUDE.md) - Go 版開発ガイド
- [@go/docs/architecture-guide.md](architecture-guide.md) - アーキテクチャガイド (UseCase の責務サマリー、エラー型、Worker と Dispatcher)
- [@go/docs/handler-guide.md](handler-guide.md) - HTTP ハンドラーガイド (UseCase の Handler からの呼び出し方)
- [@go/docs/validation-guide.md](validation-guide.md) - バリデーションガイド (UseCase が呼ぶ Validator の実装方針)
