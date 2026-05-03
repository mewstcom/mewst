# バリデーションガイド

このドキュメントは、Go 版 Mewst でのバリデーションのベストプラクティスを説明します。

## 概要

フォームからの入力値の検証は、`internal/validator/` パッケージにバリデーターとして実装します。形式バリデーション（入力値の形式チェック）と状態バリデーション（DB を使った検証）を同じバリデーターに配置することで、「どこに書くべきか」の判断コストを削減します。

### ファイル構成

```
internal/handler/sign_in/
├── handler.go         # Handler構造体と依存性
├── new.go             # フォーム表示
└── create.go          # 作成処理

internal/validator/
├── sign_in.go         # サインインのバリデーション（形式チェック + DBを使った検証）
└── sign_in_test.go    # サインインのバリデーションテスト
```

### バリデーションの分類

バリデーションは以下の 2 種類に分類されますが、同じバリデーターに実装します：

| 種類               | 責務                  | 特徴            |
| ------------------ | --------------------- | --------------- |
| 形式バリデーション | 入力値の形式チェック  | DB アクセス不要 |
| 状態バリデーション | DB の状態を使った検証 | DB アクセス必要 |

### 構造体の命名規則

- **命名規則**: `{Handler}{Action}Validator`（例: `SignInCreateValidator`, `PasswordUpdateValidator`）
- **コンストラクタ**: `New{Handler}{Action}Validator()`（例: `NewSignInCreateValidator()`）
- **入力**: `{Handler}{Action}ValidatorInput` 構造体
- **戻り値**: Go の慣習に従った `(data, error)` の 2 値返し。データを返す必要がない場合は `error` のみ
  - データ不要 → `error` のみ
  - 単一モデルを返す → `(*model.X, error)` 直接（例: `SignInCreateValidator` → `(*model.User, error)`）
  - 複数フィールドを返す → `(*{Handler}{Action}ValidateOutput, error)`
- **1 つの構造体で両方のバリデーションを担当**: 形式バリデーションと状態バリデーションを `Validate` メソッド内で順次実行
- **`main.go` で構築し UseCase に注入**: UseCase がバリデーション → 永続化をオーケストレーションする

### 状態バリデーションの配置場所

状態バリデーションは Validator または UseCase のどちらかに配置します。

**判断基準**: **「検証失敗時に DB を更新する必要があるか？」**

| 検証失敗時の DB 更新 | 配置場所  | 理由                                               |
| -------------------- | --------- | -------------------------------------------------- |
| 不要                 | Validator | UseCase をシンプルに保つため                       |
| 必要                 | UseCase   | トランザクション内で検証と更新を行う必要があるため |

**Validator で行うべき検証**:

| 検証内容                   | 失敗時の動作     | 理由                   |
| -------------------------- | ---------------- | ---------------------- |
| ユーザー存在チェック       | エラーメッセージ | DB 更新なし            |
| メールアドレス重複チェック | エラーメッセージ | DB 更新なし            |
| アットネーム重複チェック   | エラーメッセージ | DB 更新なし            |
| メール確認完了チェック     | エラーメッセージ | DB 更新なし            |
| コード一致チェック         | エラーメッセージ | DB 更新なし（※注参照） |
| パスワード照合             | エラーメッセージ | DB 更新なし            |

※注: コード検証で「試行回数インクリメント」が必要な場合は UseCase で行う

**UseCase で行うべき検証**:

| 検証内容           | 失敗時の動作           | 理由                   |
| ------------------ | ---------------------- | ---------------------- |
| ログインコード検証 | 試行回数インクリメント | 失敗時に DB 更新が必要 |

※注: 「リカバリーコード消費」や「トークン使用済みマーク」は検証成功後の処理であり、バリデーションではなく UseCase の永続化処理として扱う。検証自体は Validator で行い、成功後に UseCase を呼び出す。

### エラー表示方法の使い分け

| エラー種類           | 表示方法                 | 使い分け                                           |
| -------------------- | ------------------------ | -------------------------------------------------- |
| **フィールドエラー** | `ValidationError.Fields` | 特定の入力フィールドに関連するエラー               |
| **グローバルエラー** | `ValidationError.Global` | フォーム全体に関連するエラー（同じページに留まる） |
| **Flash メッセージ** | `session.Flash`          | リダイレクト後に表示するメッセージ（成功/エラー）  |
| **ログのみ**         | `slog.Error`             | 開発者向け情報（ユーザーには一般メッセージを表示） |

**判断フローチャート**:

```
フォームを再表示する？
├─ Yes → ValidationError（Fields または Global）
│    └─ 特定フィールドに関連？ → AddField（例: ユーザー名重複）
│    └─ フォーム全体に関連？  → AddGlobal（例: 確認コード不一致）
└─ No（リダイレクトする）→ Flash
     └─ 成功 → FlashSuccess
     └─ エラー → FlashError
```

### メッセージの国際化

バリデーションメッセージは必ず `i18n.T(ctx, "message_id")` を使用します。

## 実装例

実装例は戻り値パターンの 3 分類で示します。

### データを返さないバリデーター（`error` のみ）

DB を使った検証が不要な場合や、検証成功時にデータを返す必要がない場合は `error` のみを返します。

```go
// internal/validator/password_reset.go
package validator

import (
    "context"
    "net/mail"

    "github.com/mewstcom/mewst/go/internal/i18n"
    "github.com/mewstcom/mewst/go/internal/model"
)

// PasswordResetCreateValidator はパスワードリセット申請のバリデーションを行う
type PasswordResetCreateValidator struct{}

// NewPasswordResetCreateValidator は PasswordResetCreateValidator を生成する
func NewPasswordResetCreateValidator() *PasswordResetCreateValidator {
    return &PasswordResetCreateValidator{}
}

// PasswordResetCreateValidatorInput はバリデーションの入力パラメータ
type PasswordResetCreateValidatorInput struct {
    Email string
}

// Validate はバリデーションを行い、失敗時は *model.ValidationError を返す
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) error {
    ve := model.NewValidationError()

    // 必須チェック
    if input.Email == "" {
        ve.AddField("email", i18n.T(ctx, "error_required"))
        return ve
    }

    // フォーマットチェック
    if _, err := mail.ParseAddress(input.Email); err != nil {
        ve.AddField("email", i18n.T(ctx, "error_invalid_email"))
    }

    if ve.HasErrors() {
        return ve
    }

    return nil
}
```

### 状態バリデーションで取得した単一モデルを返すバリデーター（`(*model.X, error)`）

状態バリデーションの過程で取得したモデルを戻り値として返すと、UseCase 内でデータを二重に取得する必要がなくなります。出力が単一のモデルだけで済む場合は、Output 構造体を作らずモデルを直接返します。

```go
// internal/validator/sign_in.go
package validator

import (
    "context"
    "net/mail"

    "github.com/mewstcom/mewst/go/internal/auth"
    "github.com/mewstcom/mewst/go/internal/i18n"
    "github.com/mewstcom/mewst/go/internal/model"
    "github.com/mewstcom/mewst/go/internal/repository"
)

// SignInCreateValidator はサインインのバリデーションを行う
type SignInCreateValidator struct {
    userRepo *repository.UserRepository
}

// NewSignInCreateValidator は SignInCreateValidator を生成する
func NewSignInCreateValidator(userRepo *repository.UserRepository) *SignInCreateValidator {
    return &SignInCreateValidator{
        userRepo: userRepo,
    }
}

// SignInCreateValidatorInput はバリデーションの入力パラメータ
type SignInCreateValidatorInput struct {
    Email    string
    Password string
}

// Validate はバリデーションを行い、成功時はユーザーを返す
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*model.User, error) {
    ve := model.NewValidationError()

    // 1. 形式バリデーション
    v.validateEmail(ctx, ve, input.Email)
    v.validatePassword(ctx, ve, input.Password)

    if ve.HasErrors() {
        return nil, ve
    }

    // 2. 状態バリデーション（DB検証）
    user, err := v.userRepo.FindByEmail(ctx, input.Email)
    if err != nil {
        return nil, err
    }
    // セキュリティ対策: 存在しないメールアドレスでも同じエラーメッセージを表示
    if user == nil {
        ve.AddGlobal(i18n.T(ctx, "error_invalid_credentials"))
        return nil, ve
    }

    // パスワードを検証
    if err := auth.CheckPassword(user.PasswordDigest, input.Password); err != nil {
        ve.AddGlobal(i18n.T(ctx, "error_invalid_credentials"))
        return nil, ve
    }

    return user, nil
}

// validateEmail はメールアドレスの形式バリデーションを行う
func (v *SignInCreateValidator) validateEmail(ctx context.Context, ve *model.ValidationError, email string) {
    if email == "" {
        ve.AddField("email", i18n.T(ctx, "error_required"))
        return
    }

    if _, err := mail.ParseAddress(email); err != nil {
        ve.AddField("email", i18n.T(ctx, "error_invalid_email"))
        return
    }
}

// validatePassword はパスワードの形式バリデーションを行う
func (v *SignInCreateValidator) validatePassword(ctx context.Context, ve *model.ValidationError, password string) {
    if password == "" {
        ve.AddField("password", i18n.T(ctx, "error_required"))
        return
    }
}
```

### 複数フィールドを返すバリデーター（`(*{Handler}{Action}ValidateOutput, error)`）

検証成功時に複数のデータを返したい場合は専用の `{Handler}{Action}ValidateOutput` 構造体を定義します（命名は `Validator` ではなく `Validate` + `Output`）。Mewst には現状該当する validator はありませんが、将来 2FA 等で必要になった場合に以下のパターンで追加します。

```go
// SignInCreateValidateOutput はバリデーション成功時の出力（複数フィールドが必要な場合の例）
type SignInCreateValidateOutput struct {
    User              *model.User
    UserTwoFactorAuth *model.UserTwoFactorAuth
}

// Validate はバリデーションを行い、成功時はユーザーと 2FA 情報を返す
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*SignInCreateValidateOutput, error) {
    // ...バリデーション処理...

    return &SignInCreateValidateOutput{
        User:              user,
        UserTwoFactorAuth: twoFactor,
    }, nil
}
```

### 複数フィールドだが Output を作らない場合（複数チェックの validator）

入力に対して複数のフィールドを横断的にチェックするだけで成功時にデータを返さない場合は、`error` のみで十分です。

以下は複数フィールドの組み合わせをチェックする validator の架空の例です（Mewst の現行 `PasswordUpdateValidator` は単一フィールドのみのため、教育用の例として `Password` + `PasswordConfirmation` の 2 フィールドを持つ架空の `PasswordChangeFormValidator` を示します）。

```go
package validator

import (
    "context"

    "github.com/mewstcom/mewst/go/internal/i18n"
    "github.com/mewstcom/mewst/go/internal/model"
)

// PasswordChangeFormValidator はパスワード変更フォームのバリデーションを行う（架空の例）
type PasswordChangeFormValidator struct{}

// NewPasswordChangeFormValidator は PasswordChangeFormValidator を生成する
func NewPasswordChangeFormValidator() *PasswordChangeFormValidator {
    return &PasswordChangeFormValidator{}
}

// PasswordChangeFormValidatorInput はバリデーションの入力パラメータ
type PasswordChangeFormValidatorInput struct {
    Password             string
    PasswordConfirmation string
}

// Validate はバリデーションを行い、失敗時は *model.ValidationError を返す
func (v *PasswordChangeFormValidator) Validate(ctx context.Context, input PasswordChangeFormValidatorInput) error {
    ve := model.NewValidationError()

    // 必須チェック
    if input.Password == "" {
        ve.AddField("password", i18n.T(ctx, "error_required"))
    }

    if input.PasswordConfirmation == "" {
        ve.AddField("password_confirmation", i18n.T(ctx, "error_required"))
    }

    // 文字数チェック（rune 数で計測）
    if len(input.Password) > 0 && len([]rune(input.Password)) < 8 {
        ve.AddField("password", i18n.T(ctx, "error_password_too_short"))
    }

    // パスワード一致チェック
    if input.Password != "" && input.PasswordConfirmation != "" && input.Password != input.PasswordConfirmation {
        ve.AddField("password_confirmation", i18n.T(ctx, "error_password_mismatch"))
    }

    if ve.HasErrors() {
        return ve
    }

    return nil
}
```

## ハンドラーでの使用

### 基本パターン

Handler は Validator を直接呼び出さず、UseCase 経由でバリデーションを実行します。UseCase が `*model.ValidationError` を返した場合、Handler はフォームを再表示します。

```go
// internal/handler/sign_in/create.go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // 1. フォームデータの取得
    email := r.FormValue("email")
    password := r.FormValue("password")
    backURL := r.FormValue("back")

    // 2. UseCase を実行（バリデーション込み）
    output, err := h.signInUC.Execute(ctx, usecase.CreateSignInInput{
        Email:    email,
        Password: password,
    })
    if err != nil {
        if ve := model.AsValidationError(err); ve != nil {
            // バリデーションエラー → フォームを再表示
            w.WriteHeader(http.StatusUnprocessableEntity)
            h.renderSignInForm(w, r, ve, email, backURL)
            return
        }
        // システムエラー → 500
        slog.ErrorContext(ctx, "サインイン処理に失敗", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }

    // 3. 成功 → セッションクッキーを設定してリダイレクト
    h.sessionMgr.SetSessionCookie(w, output.Token)
    http.Redirect(w, r, redirect.GetSafeRedirectURL(backURL), http.StatusFound)
}
```

### Handler の依存性

Handler は UseCase のみに依存します。Validator は `main.go` で構築し UseCase に注入されるため、Handler は Validator を直接 import しません。

```go
// main.go
signInValidator := validator.NewSignInCreateValidator(userRepo)
createSignInUC := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)
turnstileClient := turnstile.NewClient(cfg.TurnstileSecretKey)
signInHandler := sign_in.NewHandler(cfg, sessionMgr, flashMgr, createSignInUC, turnstileClient)
```

```go
// internal/handler/sign_in/handler.go
type Handler struct {
    cfg               *config.Config
    sessionMgr        *session.Manager
    flashMgr          *session.FlashManager
    signInUC          *usecase.CreateSignInUsecase  // UseCase のみに依存
    turnstileVerifier turnstile.Verifier
}

func NewHandler(
    cfg *config.Config,
    sessionMgr *session.Manager,
    flashMgr *session.FlashManager,
    signInUC *usecase.CreateSignInUsecase,
    turnstileVerifier turnstile.Verifier,
) *Handler {
    return &Handler{
        cfg:               cfg,
        sessionMgr:        sessionMgr,
        flashMgr:          flashMgr,
        signInUC:          signInUC,
        turnstileVerifier: turnstileVerifier,
    }
}
```

## テスト

### テスト方針

バリデーションのテストは `internal/validator/` パッケージ内のテストファイルに実装します。

| テスト対象           | ファイル                             | 特徴                                   |
| -------------------- | ------------------------------------ | -------------------------------------- |
| バリデーション全体   | `internal/validator/sign_in_test.go` | 形式・状態バリデーションを統合テスト   |
| ハンドラーの振る舞い | `handler_test.go`                    | E2E テスト、正常系・代表的な異常系のみ |

**理由**:

- **シンプルな構成**: バリデーターごとに 1 つのテストファイルにテストを集約
- **問題の特定**: テスト失敗時にどの検証の問題か即座に分かる
- **保守性向上**: テストファイルの管理が容易

### バリデーションのテスト

```go
// internal/validator/sign_in_test.go
func TestSignInCreateValidator_Validate(t *testing.T) {
    // 形式バリデーションのテスト
    t.Run("形式バリデーション", func(t *testing.T) {
        tests := []struct {
            name          string
            input         SignInCreateValidatorInput
            wantErrors    bool
            expectedField string
        }{
            {
                name: "有効な入力",
                input: SignInCreateValidatorInput{
                    Email:    "user@example.com",
                    Password: "password123",
                },
                wantErrors: false,
            },
            {
                name: "メールアドレスが空",
                input: SignInCreateValidatorInput{
                    Email:    "",
                    Password: "password123",
                },
                wantErrors:    true,
                expectedField: "email",
            },
            {
                name: "パスワードが空",
                input: SignInCreateValidatorInput{
                    Email:    "user@example.com",
                    Password: "",
                },
                wantErrors:    true,
                expectedField: "password",
            },
        }

        // DBアクセスなしでテスト（形式バリデーションのみ）
        v := NewSignInCreateValidator(nil)

        for _, tt := range tests {
            t.Run(tt.name, func(t *testing.T) {
                ctx := context.Background()
                ctx = i18n.SetLocale(ctx, "ja")

                _, err := v.Validate(ctx, tt.input)

                if tt.wantErrors {
                    ve := model.AsValidationError(err)
                    if ve == nil {
                        t.Error("expected validation error, got none")
                    }
                    if tt.expectedField != "" && ve != nil && !ve.HasFieldError(tt.expectedField) {
                        t.Errorf("expected field error for %q", tt.expectedField)
                    }
                } else if tt.input.Email != "" && tt.input.Password != "" {
                    // 形式バリデーションのみの場合、DBアクセスでエラーになるのでスキップ
                }
            })
        }
    })

    // 状態バリデーションのテスト（DB必要）
    t.Run("状態バリデーション", func(t *testing.T) {
        // 共有DB接続プールからトランザクションをセットアップ
        _, tx := testutil.SetupTestDB(t)

        // テストユーザーを作成（auth.HashPassword でダイジェスト化してから渡す）
        passwordDigest, _ := auth.HashPassword("password123")
        testutil.NewUserBuilder(t, tx).
            WithEmail("test@example.com").
            WithPasswordDigest(passwordDigest).
            Build()

        userRepo := repository.NewUserRepository(testutil.QueriesWithTx(tx))
        v := NewSignInCreateValidator(userRepo)

        t.Run("有効な認証情報", func(t *testing.T) {
            ctx := context.Background()
            input := SignInCreateValidatorInput{
                Email:    "test@example.com",
                Password: "password123",
            }

            user, err := v.Validate(ctx, input)

            if err != nil {
                t.Errorf("unexpected error: %v", err)
            }
            if user == nil {
                t.Error("expected user, got nil")
            }
        })

        t.Run("無効なパスワード", func(t *testing.T) {
            ctx := context.Background()
            input := SignInCreateValidatorInput{
                Email:    "test@example.com",
                Password: "wrongpassword",
            }

            user, err := v.Validate(ctx, input)

            if user != nil {
                t.Error("expected nil user")
            }
            if ve := model.AsValidationError(err); ve == nil {
                t.Error("expected validation error")
            }
        })
    })
}
```

### 正規表現のテスト

```go
func TestEmailRegex(t *testing.T) {
    tests := []struct {
        email string
        valid bool
    }{
        {"user@example.com", true},
        {"user.name@example.co.jp", true},
        {"user+tag@example.com", true},
        {"", false},
        {"invalid", false},
        {"@example.com", false},
        {"user@", false},
        {"user@.com", false},
    }

    for _, tt := range tests {
        t.Run(tt.email, func(t *testing.T) {
            if got := emailRegex.MatchString(tt.email); got != tt.valid {
                t.Errorf("emailRegex.MatchString(%q) = %v, want %v", tt.email, got, tt.valid)
            }
        })
    }
}
```

## ベストプラクティス

### 1. バリデーションは 1 つのバリデーターに統合

```go
// ✅ Good: 1つのバリデーターに形式・状態バリデーションを統合
// internal/validator/sign_in.go
type SignInCreateValidator struct {
    userRepo *repository.UserRepository
}

func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*model.User, error) {
    // 1. 形式バリデーション（DB不要）
    ve := model.NewValidationError()
    // ...

    // 2. 状態バリデーション（DB必要）
    user, err := v.userRepo.FindByEmail(ctx, input.Email)
    // ...
    return user, nil
}
```

### 2. 国際化を徹底

```go
// ❌ Bad: ハードコードされたメッセージ
ve.AddField("email", "メールアドレスを入力してください")

// ✅ Good: 国際化された翻訳
ve.AddField("email", i18n.T(ctx, "error_required"))
```

### 3. 早期リターンでネストを減らす

```go
// ❌ Bad: ネストが深い
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) error {
    ve := model.NewValidationError()
    if input.Email != "" {
        if emailRegex.MatchString(input.Email) {
            // OK
        } else {
            ve.AddField("email", "...")
        }
    } else {
        ve.AddField("email", "...")
    }
    if ve.HasErrors() {
        return ve
    }
    return nil
}

// ✅ Good: 早期リターンでシンプル
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) error {
    ve := model.NewValidationError()

    if input.Email == "" {
        ve.AddField("email", i18n.T(ctx, "error_required"))
        return ve
    }

    if !emailRegex.MatchString(input.Email) {
        ve.AddField("email", i18n.T(ctx, "error_invalid_email_format"))
    }

    if ve.HasErrors() {
        return ve
    }

    return nil
}
```

### 4. 正規表現はパッケージレベルで定義

```go
// ✅ Good: 正規表現のコンパイルを1回だけ実行
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) error {
    // emailRegexを使用
}
```

### 5. Go の慣習に従った `(data, error)` の 2 値返し

戻り値はバリデーターが返すデータの有無で使い分けます。

```go
// ✅ Good: データを返す場合は (data, error)
// 単一モデルなら専用の Output 構造体を作らずモデル直接返し
func (v *SignInCreateValidator) Validate(ctx context.Context, input SignInCreateValidatorInput) (*model.User, error) {
    ve := model.NewValidationError()
    // ...
    if ve.HasErrors() {
        return nil, ve  // *model.ValidationError は error を満たす
    }

    user, err := v.userRepo.FindByEmail(ctx, input.Email)
    if err != nil {
        return nil, err  // システムエラー
    }

    return user, nil
}

// ✅ Good: データを返さない場合は error のみ
func (v *PasswordResetCreateValidator) Validate(ctx context.Context, input PasswordResetCreateValidatorInput) error {
    ve := model.NewValidationError()
    // ...
    if ve.HasErrors() {
        return ve
    }
    return nil
}
```

## 利点

1. **シンプルな構成**: バリデーションロジックが 1 つのバリデーターに集約される
2. **判断コストの削減**: 「どこに書くべきか」を迷わない
3. **依存が明確**: バリデーターの依存関係が一目でわかる
4. **テストしやすい**: 1 つのテストファイルでバリデーション全体をテストできる
5. **再利用可能**: 同じバリデーターを複数の UseCase で使用可能
