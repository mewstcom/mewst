# depguard ルールの Wikino 整合 作業計画書

<!--
このテンプレートの使い方:
1. このファイルを `docs/plans/2_todo/` ディレクトリにコピー
   例: cp docs/plans/template.md docs/plans/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**作業計画書の性質**:
- 作業計画書は「何をどう変えるか」という変更内容を記述するドキュメントです
- 新しい機能の場合は、概要・要件・設計もこのドキュメントに記述します
- 現在のシステムの状態は `docs/specs/` の仕様書に記述されています
- タスク完了後は、仕様書を新しい状態に更新してください（設計判断や採用しなかった方針も含める）

**仕様書との関係**:
- 新しい機能の場合: タスク完了後に `docs/specs/` に仕様書を作成する
- 既存機能の変更の場合: 「仕様書」セクションに対応する仕様書へのリンクを記載し、タスク完了後に仕様書を更新する

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 仕様書

- [アーキテクチャガイド](/workspace/go/docs/architecture-guide.md)
- [golangci-lint 設定](/workspace/go/.golangci.yml)

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

Mewst の `.golangci.yml` の depguard 設定を Wikino と整合させるリファクタリング。Wikino で先行して確立したアーキテクチャルールを Mewst にも適用することで、両プロジェクト間のアーキテクチャの一貫性を維持する。

### 変更の背景

Wikino では UseCase オーケストレーションリファクタリング（[作業計画書](/wikino/docs/plans/1_doing/usecase-orchestration-refactor.md)）を進行中であり、以下のアーキテクチャ変更が行われている：

- **Worker を Presentation 層に位置づけ**: UseCase を呼ぶだけの薄い Adapter にする
- **Dispatcher パッケージの新設**: ジョブキューへの投入を抽象化し、UseCase ↔ Worker 間の循環依存を解消
- **Handler → Validator の直接依存を禁止**: バリデーションを UseCase 内で実行するように統一
- **UseCase → session の依存を禁止**: session はPresentation層のヘルパーとして位置づけ

これらの変更に伴い、depguard ルールに差分が生じている。Mewst でも同じルールを導入し、段階的にコードを修正する。

### Mewst と Wikino の depguard 差分

| レイヤー                 | 変更内容                                                        | 既存コード違反                                                     |
| ------------------------ | --------------------------------------------------------------- | ------------------------------------------------------------------ |
| application-layer        | `templates` deny を維持（email パッケージがレンダリングを担当） | `send_email_confirmation.go` が templates を import（4-1a で解消） |
| application-layer        | `session` deny を追加                                           | `create_session.go` が `session.GenerateToken()` を使用            |
| handler-layer            | `validator` deny を追加                                         | 多数のハンドラーが validator を直接 import                         |
| worker-layer（新設）     | query, handler, middleware, viewmodel, templates を deny        | `send_email_confirmation.go` が templates を import                |
| email-layer（新設）      | handler, usecase, worker 等の上位層・同レイヤーへの依存を禁止   | なし                                                               |
| dispatcher-layer（新設） | 上位層・同レイヤーへの依存を禁止                                | パッケージ未存在                                                   |

**対象外**: Policy 層と Markup 層は Mewst に存在しないため対象外。

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

- depguard ルールを Wikino と同一にする（Policy 層・Markup 層を除く）
- Worker 層・Dispatcher 層の依存関係ルールを新設する
- `model.ValidationError` と `model.AppError` を `internal/model/` に定義する
- Validator の Result 型を廃止し、Go の慣習に従った `(data, error)` の2値返しに変更する
- `session.FormErrors` を `model.ValidationError` に置き換え、最終的に `session.FormErrors` を削除する
- Handler は `errors.As` パターンで UseCase からのエラー（`ValidationError` / `AppError` / 素の `error`）を判別する
- 既存コードが新ルールに違反しないようにリファクタリングする
- 外部からの振る舞いは変わらない（リファクタリングのみ）

### 非機能要件

- **一貫性**: Wikino と Mewst で同一の depguard ルールを使用する
- **段階的移行**: ルール追加とコード修正をフェーズに分け、各フェーズで `make lint` が通る状態を維持する

## 実装ガイドラインの参照

<!--
**重要**: 作業計画書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
作業計画書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン（**ファイル名は標準の9種類のみ**）
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

### 参考: Wikino の設定

- [Wikino golangci-lint 設定](/wikino/go/.golangci.yml) - 目標とする depguard ルール
- [Wikino アーキテクチャガイド](/wikino/go/docs/architecture-guide.md) - Worker・Dispatcher の設計
- [Wikino UseCase オーケストレーション リファクタリング](/wikino/docs/plans/1_doing/usecase-orchestration-refactor.md) - Handler → Validator 禁止の経緯

## 設計

<!--
ガイドライン:
- 技術的な実装の設計を記述

**重要: 設計は実装中に更新する**:
- 作業計画書内の設計は初期の方針であり、完璧ではない
- 実装中により良いアプローチが見つかった場合は、設計を積極的に更新する
- 設計に固執して実装の質を下げるよりも、実装で得た知見を設計に反映する方が重要
- 変更した場合は「採用しなかった方針」セクションに変更前の方針と変更理由を記録する
-->

### depguard ルール変更一覧

#### 既存レイヤーの変更

**application-layer**:

```yaml
# 変更前
application-layer:
  files:
    - "**/internal/usecase/**"
  deny:
    - pkg: .../internal/query
    - pkg: .../internal/handler
    - pkg: .../internal/middleware
    - pkg: .../internal/viewmodel
    - pkg: .../internal/templates  # ← 維持（削除しない）

# 変更後
application-layer:
  files:
    - "**/internal/usecase/**"
  deny:
    - pkg: .../internal/query
    - pkg: .../internal/handler
    - pkg: .../internal/middleware
    - pkg: .../internal/viewmodel
    - pkg: .../internal/templates    # ← 維持（email パッケージがレンダリングを担当）
    - pkg: .../internal/session      # ← 追加
```

**handler-layer**:

```yaml
# 変更前
handler-layer:
  files:
    - "**/internal/handler/**"
  deny:
    - pkg: .../internal/query
    - pkg: .../internal/repository

# 変更後
handler-layer:
  files:
    - "**/internal/handler/**"
  deny:
    - pkg: .../internal/query
    - pkg: .../internal/repository
    - pkg: .../internal/validator    # ← 追加
```

#### 新規レイヤー

**worker-layer**:

```yaml
worker-layer:
  files:
    - "**/internal/worker/**"
  deny:
    - pkg: .../internal/query
      desc: "WorkerはQueryに直接依存できません。Repositoryを経由してください。"
    - pkg: .../internal/handler
      desc: "WorkerはPresentation層に依存できません。"
    - pkg: .../internal/middleware
      desc: "WorkerはPresentation層に依存できません。"
    - pkg: .../internal/viewmodel
      desc: "WorkerはPresentation層に依存できません。"
    - pkg: .../internal/templates
      desc: "WorkerはPresentation層に依存できません。UseCaseを経由してください。"
```

**email-layer**（新設）:

```yaml
email-layer:
  files:
    - "**/internal/email/**"
  deny:
    - pkg: .../internal/query
      desc: "emailパッケージはQueryに依存できません。"
    - pkg: .../internal/repository
      desc: "emailパッケージはRepositoryに依存できません。"
    - pkg: .../internal/handler
      desc: "emailパッケージはHandlerに依存できません。"
    - pkg: .../internal/middleware
      desc: "emailパッケージはMiddlewareに依存できません。"
    - pkg: .../internal/viewmodel
      desc: "emailパッケージはViewModelに依存できません。"
    - pkg: .../internal/usecase
      desc: "emailパッケージはUseCaseに依存できません。"
    - pkg: .../internal/validator
      desc: "emailパッケージはValidatorに依存できません。"
    - pkg: .../internal/worker
      desc: "emailパッケージはWorkerに依存できません。"
    - pkg: .../internal/dispatcher
      desc: "emailパッケージはDispatcherに依存できません。"
    - pkg: .../internal/session
      desc: "emailパッケージはSessionに依存できません。"
```

許可される依存先: `templates`, `i18n`, `model`, `config`, `auth`, 外部パッケージ（`templ`, `resend`）

**dispatcher-layer**:

```yaml
dispatcher-layer:
  files:
    - "**/internal/dispatcher/**"
  deny:
    - pkg: .../internal/usecase
      desc: "DispatcherはApplication層に依存できません。"
    - pkg: .../internal/validator
      desc: "DispatcherはApplication層に依存できません。"
    - pkg: .../internal/worker
      desc: "DispatcherはWorkerに依存できません。"
    - pkg: .../internal/handler
      desc: "DispatcherはPresentation層に依存できません。"
    - pkg: .../internal/middleware
      desc: "DispatcherはPresentation層に依存できません。"
    - pkg: .../internal/viewmodel
      desc: "DispatcherはPresentation層に依存できません。"
    - pkg: .../internal/templates
      desc: "DispatcherはPresentation層に依存できません。"
    - pkg: .../internal/query
      desc: "DispatcherはQueryに依存できません。"
    - pkg: .../internal/repository
      desc: "DispatcherはRepositoryに依存できません。"
    - pkg: .../internal/model
      desc: "DispatcherはModelに依存できません。"
```

### 既存コードの違反と修正方針

#### 1. UseCase → session の依存（`create_session.go`）

**現状**: `create_session.go` が `session.GenerateToken()` を呼び出している。

**修正方針**: `GenerateToken()` 関数を `internal/auth/` パッケージに移動する。セッショントークンの生成は認証ロジックの一部であり、`auth` パッケージが適切。Wikino でも同じ判断がされている（depguard の desc に「トークン生成はauthパッケージを使用してください」と記載）。

```go
// 変更前: internal/usecase/create_session.go
import "github.com/mewstcom/mewst/go/internal/session"
token, err := session.GenerateToken()

// 変更後
import "github.com/mewstcom/mewst/go/internal/auth"
token, err := auth.GenerateSecureToken()
```

#### 2. Worker → templates の依存（`send_email_confirmation.go`）

**現状**: `send_email_confirmation.go` が `internal/templates/emails/email_confirmation` を import してメールの HTML レンダリングを行っている。

**修正方針**: メール送信 UseCase を新設し、Worker は UseCase を呼ぶだけの薄い Adapter にする。テンプレートレンダリングは email パッケージの `ConfirmationSender` に移動し、UseCase は interface 経由で呼び出すことで `internal/templates` への直接依存を避ける。

```go
// 変更前: Worker がテンプレートを直接レンダリング
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[...]) error {
    htmlBody := renderEmailTemplate(ctx, job.Args)
    w.sender.SendRaw(ctx, ...)
}

// 変更後: Worker は UseCase を呼ぶだけ
func (w *SendEmailConfirmationWorker) Work(ctx context.Context, job *river.Job[...]) error {
    return w.sendEmailConfirmationUC.Execute(ctx, usecase.SendEmailConfirmationInput{...})
}
```

#### 2a. email パッケージのテンプレートレンダリング責務

**現状**: `email.Sender` インターフェースに `Send`（`templ.Component` ベース）と `SendRaw`（文字列ベース）の 2 メソッドが存在する。UseCase が `SendRaw` を使用してテンプレートを自前でレンダリングしているが、`SendRaw` は Worker 向け API（コメントにも「Worker用」と記載）であり、UseCase が使うのは設計意図に反する。

**修正方針**: email パッケージに `ConfirmationSender` を新設し、テンプレートレンダリング + i18n（件名取得）を email パッケージ側に閉じ込める。UseCase は自前で定義した小さい interface に依存するだけで、`internal/templates` を import しない。

```go
// internal/email/confirmation.go（email パッケージ、Presentation 層）
type ConfirmationSender struct {
    sender Sender
}

func (s *ConfirmationSender) Send(ctx context.Context, to, code, locale string) error {
    ctx = i18n.SetLocale(ctx, locale)
    subject := i18n.T(ctx, "email_confirmation_subject")

    var htmlBody, textBody templ.Component
    switch locale {
    case "en":
        htmlBody = email_confirmation.EnHTML(to, code)
        textBody = email_confirmation.EnText(to, code)
    default:
        htmlBody = email_confirmation.JaHTML(to, code)
        textBody = email_confirmation.JaText(to, code)
    }

    return s.sender.Send(ctx, SendInput{
        To: to, Subject: subject, HTMLBody: htmlBody, TextBody: textBody,
    })
}
```

```go
// internal/usecase/send_email_confirmation.go（UseCase、Application 層）
// UseCase 側で interface を定義（templates に依存しない）
type EmailConfirmationSender interface {
    Send(ctx context.Context, to, code, locale string) error
}

type SendEmailConfirmationUsecase struct {
    sender EmailConfirmationSender
}

func (uc *SendEmailConfirmationUsecase) Execute(ctx context.Context, input SendEmailConfirmationInput) error {
    if input.Email == "" {
        return fmt.Errorf("メールアドレスが空です")
    }
    if err := uc.sender.Send(ctx, input.Email, input.Code, input.Locale); err != nil {
        slog.ErrorContext(ctx, "メール確認コードの送信に失敗", "error", err, "email", input.Email)
        return fmt.Errorf("メール確認コードの送信に失敗: %w", err)
    }
    slog.InfoContext(ctx, "メール確認コードを送信しました", "email", input.Email)
    return nil
}
```

**`SendRaw` を削除する理由**:

- `SendRaw` の唯一の利用者は `SendEmailWorker`（汎用メール送信 Worker）だが、`EnqueueEmail` を呼び出す UseCase が現時点で存在せず、未使用の Infrastructure である
- YAGNI 原則に従い、未使用の `SendRaw` / `SendRawInput` / `SendEmailWorker` / `SendEmailArgs` / `EnqueueEmail` をまとめて削除する
- `Sender` インターフェースは `Send`（`templ.Component` ベース）のみになる
- 将来汎用メール送信が必要になった場合は、その時点で型固有の Sender（`ConfirmationSender` と同じパターン）を作成する

**email パッケージの層の位置づけ**:

email パッケージは Infrastructure（API 送信）と Presentation（テンプレートレンダリング）の両面を持つクロスカッティングなパッケージ。depguard ルールによる制約は設けず、必要に応じて `internal/templates` や `internal/i18n` を import する。

#### 3. Dispatcher パッケージの新設

**現状**: パッケージが存在しない。Worker の Args 型は `internal/worker/` に定義されている。

**修正方針**: Wikino に倣い `internal/dispatcher/` を新設。Args 型と Enqueue メソッドをここに配置し、UseCase ↔ Worker 間の循環依存を解消する。

```
依存の方向:
Worker (Presentation)     → dispatcher + usecase
UseCase (Application)     → dispatcher
Dispatcher (Domain/Infra) → river（外部ライブラリのみ）
```

#### 4. Handler → Validator の依存

**現状**: 多数のハンドラーが `internal/validator/` を直接 import している。

**修正方針**: Wikino の UseCase オーケストレーションリファクタリングに倣い、バリデーションの呼び出しを UseCase に移動する。Handler は UseCase のみを呼び出し、UseCase が内部で Validator を使用する。これは最も影響範囲が大きい変更であり、最終フェーズで実施する。

### エラー型の設計

Wikino の UseCase オーケストレーションリファクタリングで確立されたエラー型パターンを Mewst にも導入する。

#### エラー型の定義

`internal/model/errors.go` に以下のエラー型を定義する：

**ValidationError**: バリデーションエラー（ユーザーが入力を修正できるエラー）。Handler はフォームを再描画する。

**AppError**: アプリケーションエラー（[SafeError パターン](https://blog.jetbrains.com/go/2026/03/02/secure-go-error-handling-best-practices/)を参考）。`Error()` メソッドはユーザー安全なメッセージのみを返し、内部エラーの露出を構造的に防止する。

```go
// internal/model/errors.go

// ValidationError はバリデーションエラーを表す。
// Handler はこのエラーを受け取ったらフォームを再描画する（422）。
type ValidationError struct {
    Global []string
    Fields map[string][]string
}

func (e *ValidationError) Error() string { return "validation failed" }

func (e *ValidationError) AddGlobal(message string) { ... }
func (e *ValidationError) AddField(field, message string) { ... }
func (e *ValidationError) HasErrors() bool { ... }

// AppErrorCode はアプリケーションエラーの種別を表す型
type AppErrorCode int

const (
    AppErrCodeResourceNotFound AppErrorCode = iota + 1
    AppErrCodeForbidden
    AppErrCodeConflict
    AppErrCodeInternal
)

// AppError はアプリケーションエラーを表す（SafeError パターン）。
// Error() はユーザー安全なメッセージのみを返す。
type AppError struct {
    Code     AppErrorCode
    UserMsg  string
    Internal error
    Metadata map[string]string
}

func (e *AppError) Error() string { return e.UserMsg }
```

#### エラー型の使い分け

| エラー型                 | 生成元    | 意味                             | Handler の対応                          |
| ------------------------ | --------- | -------------------------------- | --------------------------------------- |
| `*model.ValidationError` | Validator | 入力が不正（ユーザーが修正可能） | フォーム再描画（422）                   |
| `*model.AppError`        | UseCase   | 業務レベルの既知の失敗           | エラーコードに応じた処理（403, 404 等） |
| 素の `error`             | どこでも  | 予期しないシステムエラー         | 500                                     |

#### 依存の方向

```
Handler (Presentation) → errors.As で ValidationError / AppError を判別
UseCase (Application)  → ValidationError / AppError を return
Validator (Application) → ValidationError を生成
Model (Domain/Infra)    → ValidationError / AppError を定義
```

#### Handler での使用パターン

```go
output, err := h.createSessionUC.Execute(ctx, input)
if err != nil {
    var ve *model.ValidationError
    if errors.As(err, &ve) {
        w.WriteHeader(http.StatusUnprocessableEntity)
        h.renderForm(w, r, ..., ve, ...)
        return
    }
    var ae *model.AppError
    if errors.As(err, &ae) {
        slog.ErrorContext(ctx, ae.LogString())
        // ae.Error() は安全なメッセージのみ返す
    }
    slog.ErrorContext(ctx, "予期しないエラー", "error", err)
    http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    return
}
```

#### Validator の変更

Result 型を廃止し、Go の慣習に従った `(data, error)` の2値返しに変更する。

```go
// 変更前
type SignInCreateValidatorResult struct {
    User       *model.User
    FormErrors *session.FormErrors
    Err        error
}
func (v *SignInCreateValidator) Validate(ctx, input) *SignInCreateValidatorResult

// 変更後
type SignInCreateValidatorOutput struct {
    User *model.User
}
func (v *SignInCreateValidator) Validate(ctx, input) (*SignInCreateValidatorOutput, error)
// エラー時は *model.ValidationError（error を満たす）を返す
```

#### `session.FormErrors` との関係

- `session.FormErrors` は廃止し、`model.ValidationError` に置き換える
- テンプレートは `model.ValidationError` を直接受け取る（構造は同じ `Global` + `Fields`）
- `internal/session/form_errors.go` を最終的に削除する

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

### A. handler-layer の validator deny を先に追加し、lint を一時的に無視する

golangci-lint の `exclusions` でハンドラーの validator import を一時的に除外し、段階的に修正する方針を検討した。

**不採用の理由**:

- lint が通らない状態を許容すると、他の違反も見過ごされるリスクがある
- 「フェーズごとに `make lint` が通る状態を維持する」原則に反する
- コード修正が完了してから deny ルールを追加する方が安全

### B. ValidationError と AppError を Application 層に配置する

`internal/usecase/errors.go` または新設の `internal/apperror/` にエラー型を定義する方針を検討した。

**不採用の理由**:

- Validator（Application 層）が `ValidationError` を生成するために `usecase` パッケージを import すると、UseCase → Validator という依存の方向に対して Validator → UseCase の逆方向依存が発生し、循環依存のリスクがある
- 新設パッケージ（`internal/apperror/`）を作ると、パッケージが増えて複雑になる
- Model（Domain/Infrastructure 層）は依存グラフの最下層にあり、すべての層から自然に参照できるため、エラー型の配置先として適切

### C. application-layer から templates deny を削除して UseCase でテンプレートをレンダリングする

UseCase が `internal/templates` を直接 import してメールテンプレートをレンダリングする方針を検討し、Phase 1-1 で `templates` deny を削除した。

**不採用の理由**:

- UseCase（Application 層）が `internal/templates`（Presentation 層）に依存するのは、レイヤー間の依存方向に反する
- `email.Sender` インターフェースの `SendRaw` は Worker 向け API（コメントにも「Worker用」と記載）であり、UseCase が使うのは設計意図に反する
- email パッケージに `ConfirmationSender` を新設し、テンプレートレンダリングを email パッケージ側に閉じ込めることで、application-layer の `templates` deny を維持できる
- UseCase は自前で定義した小さい interface に依存するだけでよく、よりシンプルになる

### D. session.GenerateToken() を repository に移動する

セッショントークン生成を Repository に移動する方針を検討した。

**不採用の理由**:

- Repository はデータアクセスの抽象化であり、トークン生成は Repository の責務ではない
- Wikino の depguard 設定で「トークン生成はauthパッケージを使用してください」と明示されている
- `auth` パッケージが認証に関するユーティリティの適切な配置先

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- チェックボックスで進捗を管理
- **重要**: 1タスク = 1 Pull Request の粒度で作成してください
- **重要**: 各タスクには想定ファイル数と想定行数を明記してください（PRサイズの見積もりのため）
- 依存関係を明確に
- Pull Requestのガイドラインは CLAUDE.md を参照（変更ファイル数20以下、変更行数300行以下）
-->

### フェーズ 1: 即時適用可能な depguard 変更

<!--
既存コードに違反がない変更のみを先行して適用する。
-->

- [x] **1-1**: [Go] depguard ルール変更（違反なしの変更）
  - ~~application-layer から `templates` deny を削除（メールレンダリングのため許可）~~ → タスク 4-1a で再追加する
  - worker-layer を新設（`templates` deny は除外。query, handler, middleware, viewmodel を deny）
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

### フェーズ 2: UseCase → session 依存の解消

- [x] **2-1**: [Go] `session.GenerateToken()` を `auth` パッケージに移動
  - `internal/auth/token.go` に `GenerateSecureToken()` 関数を追加（Wikino と統一）
  - `internal/usecase/create_session.go` の import を `session` → `auth` に変更
  - `internal/session/` から `GenerateToken()` を削除（他に参照がなければ）
  - application-layer に `session` deny を追加
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 4 ファイル（実装 3 + テスト 1）
  - **想定行数**: 約 50 行（実装 30 行 + テスト 20 行）

### フェーズ 3: Dispatcher パッケージの新設

- [x] **3-1**: [Go] Dispatcher パッケージの作成と depguard ルール追加
  - `internal/dispatcher/dispatcher.go` を新規作成
  - 既存の Worker Args 型を `internal/worker/` から `internal/dispatcher/` に移動
  - Enqueue メソッド（`EnqueueEmailConfirmation` 等）を定義
  - UseCase の Worker 直接呼び出しを Dispatcher 経由に変更（現時点で UseCase が Worker に依存している箇所がある場合）
  - dispatcher-layer の depguard ルールを追加
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 250 行（実装 150 行 + テスト 100 行）

### フェーズ 4: Worker → templates 依存の解消

<!--
Wikino の UseCase オーケストレーションリファクタリング フェーズ 4 に対応。
メール送信 UseCase を新設し、Worker を薄い Adapter にする。
-->

- [x] **4-1**: [Go] メール送信 UseCase の新設
  - `internal/usecase/send_email_confirmation.go` を新規作成
  - テンプレートレンダリング + メール送信ロジックを Worker から UseCase に移動
  - **想定ファイル数**: 約 4 ファイル（実装 2 + テスト 2）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

- [x] **4-1a**: [Go] email パッケージへのテンプレートレンダリング移動
  - `internal/email/confirmation.go` に `ConfirmationSender` を新設（テンプレートレンダリング + i18n 件名取得）
  - `internal/usecase/send_email_confirmation.go` から `internal/templates` import を削除し、`EmailConfirmationSender` interface パターンに変更
  - application-layer に `templates` deny を再追加（Phase 1-1 で削除した分を元に戻す）
  - email-layer の depguard ルールを新設（templates, i18n 等は許可、handler, usecase 等は deny）
  - テスト更新
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 7 ファイル（実装 4 + テスト 3）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

- [x] **4-2**: [Go] Worker を薄い Adapter に変更 + SendRaw 削除
  - `send_email_confirmation.go` Worker からテンプレートレンダリングを削除し、UseCase を呼ぶだけに変更
  - `Sender` インターフェースから `SendRaw` / `SendRawInput` を削除（`Send` のみに統一）
  - 未使用の `SendEmailWorker` / `dispatcher.SendEmailArgs` / `dispatcher.EnqueueEmail` を削除
  - `NoopSender` から `SentRawEmails` フィールドを削除
  - `main.go` の DI 構成を更新（`ConfirmationSender` を構築して `SendEmailConfirmationUsecase` に注入）
  - worker-layer に `templates` deny を追加
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 10 ファイル（実装 6 + テスト 4）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行、削除行多め）

### フェーズ 5: エラー型の基盤整備

<!--
Wikino の UseCase オーケストレーションリファクタリング フェーズ 1 に対応。
UseCase オーケストレーション（フェーズ 6）の前提となるエラー型を定義する。
-->

- [x] **5-1**: [Go] エラー型の定義（`model.ValidationError`, `model.AppError`）
  - `internal/model/errors.go` を新規作成
  - `ValidationError`（Global + Fields）、`AppError`（SafeError パターン）、`AppErrorCode`（iota 定数）を定義
  - ヘルパー関数（`NewValidationError`, `NewAppError` 等）を定義
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 200 行（実装 100 行 + テスト 100 行）

### フェーズ 6: Handler → Validator 依存の解消（UseCase オーケストレーション）

<!--
Wikino の UseCase オーケストレーションリファクタリング フェーズ 2〜3 に対応。
最も影響範囲が大きい変更。Handler から Validator の直接呼び出しを削除し、
UseCase 内でバリデーション・永続化を統括するパターンに移行する。

Mewst には Policy 層が存在しないため、Wikino の認可統合部分は対象外。
バリデーションの UseCase 統合のみを実施する。

各タスクでは以下を含む：
- Validator の Result 型を廃止し (data, error) 返しに変更
- UseCase に Validator 呼び出しを統合
- Handler を errors.As パターンに変更
- テンプレートの session.FormErrors 参照を model.ValidationError に変更

対象ハンドラー:
  - sign_in/create.go
  - sign_up/create.go
  - email_confirmation/create.go
  - password/update.go
  - password_reset/create.go
  - accounts/create.go
-->

- [x] **6-1**: [Go] パイロット: sign_in UseCase にバリデーションを統合
  - `SignInCreateValidator` の Result 型を廃止し `(*SignInCreateValidatorOutput, error)` 返しに変更（`*model.ValidationError` を error として返す）
  - `internal/usecase/create_session.go`（または新規 UseCase）に Validator の呼び出しを統合
  - `internal/handler/sign_in/create.go` から Validator の直接呼び出しを削除
  - Handler は UseCase のみを呼び出し、`errors.As` パターンでエラーを判別
  - 対象テンプレートの `session.FormErrors` → `model.ValidationError` に変更
  - テストを更新
  - **想定ファイル数**: 約 10 ファイル（実装 5 + テスト 5）
  - **想定行数**: 約 400 行（実装 200 行 + テスト 200 行）

- [x] **6-2**: [Go] 残りのハンドラーの Validator 依存を解消
  - sign_up, email_confirmation, password, password_reset, accounts の各ハンドラーを修正
  - 各 Validator の Result 型を廃止し `(data, error)` 返しに変更
  - 各 UseCase にバリデーション呼び出しを統合
  - Handler から Validator の直接 import を削除し `errors.As` パターンに変更
  - 対象テンプレートの `session.FormErrors` → `model.ValidationError` に変更
  - テストを更新
  - **想定ファイル数**: 約 18 ファイル（実装 10 + テスト 8）
  - **想定行数**: 約 600 行（実装 300 行 + テスト 300 行）

- [x] **6-3**: [Go] `session.FormErrors` の完全除去
  - `internal/templates/components/form_errors.templ` を `model.ValidationError` に変更
  - 残存する全テンプレートの `session.FormErrors` 参照を除去
  - `internal/session/form_errors.go` を削除
  - `make templ-generate` を実行し、生成ファイルを更新
  - **想定ファイル数**: 約 10 ファイル（実装 10 + テスト 0）
  - **想定行数**: 約 150 行（実装 150 行 + テスト 0 行）

- [x] **6-4**: [Go] handler-layer に validator deny を追加
  - handler-layer depguard ルールに `validator` deny を追加
  - `make lint` で違反がないことを確認
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 5 行（実装 5 行 + テスト 0 行）

### フェーズ 7: ドキュメント更新

- [x] **7-1**: [Go] アーキテクチャガイドの更新
  - 3層アーキテクチャの図に Worker（Presentation 層）と Dispatcher（Domain/Infrastructure 層）を追加
  - 依存関係ルールの更新（depguard ルールとの整合性）
  - UseCase のオーケストレーション責務を反映（フェーズ 6 実施後）
  - エラー型（`ValidationError`, `AppError`）の使い分けを追記
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 150 行（実装 150 行 + テスト 0 行）

### フェーズ 8: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [x] **8-1**: 仕様書の作成・更新
  - `docs/specs/` に仕様書を作成または更新する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **Policy 層の depguard ルール**: Mewst に Policy 層が存在しないため対象外
- **Markup 層の depguard ルール**: Mewst に Markup 層が存在しないため対象外

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Wikino UseCase オーケストレーション リファクタリング 作業計画書](/wikino/docs/plans/1_doing/usecase-orchestration-refactor.md)
- [Wikino アーキテクチャガイド](/wikino/go/docs/architecture-guide.md)
- [Wikino golangci-lint 設定](/wikino/go/.golangci.yml)
