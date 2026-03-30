# コードレビュー: handler-2-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-30                                     |
| 対象ブランチ               | handler-2-1                                    |
| ベースブランチ             | handler-1-1                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 12 ファイル                                    |
| 変更行数（実装）           | +50 / -45 行                                   |
| 変更行数（テスト）         | +57 / -53 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_up/create.go`
- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/validator/sign_in.go`
- [x] `go/internal/validator/sign_up.go`

### テストファイル

- [x] `go/internal/handler/sign_in/handler_test.go`
- [x] `go/internal/handler/sign_up/handler_test.go`
- [x] `go/internal/validator/sign_in_test.go`
- [x] `go/internal/validator/sign_up_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

作業計画書タスク 2-1 の要件との整合性を確認しました。

| 要件                                                                     | 状態 |
| ------------------------------------------------------------------------ | ---- |
| `internal/validator/` パッケージを新規作成                               | ✅   |
| `handler/sign_in/validator.go` → `validator/sign_in.go` に移動・リネーム | ✅   |
| `handler/sign_up/validator.go` → `validator/sign_up.go` に移動・リネーム | ✅   |
| 対応するテストファイルも移動                                             | ✅   |
| Handler の `NewHandler` を更新し、Validator を外部から受け取るように変更 | ✅   |
| `main.go` で Validator を構築し Handler に渡すように変更                 | ✅   |

**命名規則の整合性**:

| 変更前                         | 変更後                                 | 計画書の命名 | 一致 |
| ------------------------------ | -------------------------------------- | ------------ | ---- |
| `sign_in.CreateValidator`      | `validator.SignInCreateValidator`      | 同左         | ✅   |
| `sign_in.NewCreateValidator()` | `validator.NewSignInCreateValidator()` | 同左         | ✅   |
| `sign_up.CreateValidator`      | `validator.SignUpCreateValidator`      | 同左         | ✅   |
| `sign_up.NewCreateValidator()` | `validator.NewSignUpCreateValidator()` | 同左         | ✅   |

**構築パターンの整合性**:

計画書で定義された「main.go で Validator を構築し Handler に渡す」パターンが正しく実装されています。

```go
// main.go
signInValidator := validator.NewSignInCreateValidator(userRepo)
signUpValidator := validator.NewSignUpCreateValidator(userRepo)
signInHandler := sign_in.NewHandler(cfg, sessionMgr, actorRepo, createSessionUC, turnstileClient, signInValidator)
signUpHandler := sign_up.NewHandler(cfg, sessionMgr, createEmailConfirmationUC, turnstileClient, rateLimiter, signUpValidator)
```

**既知の残存依存（計画通り）**:

`sign_in/handler.go` には `actorRepo *repository.ActorRepository` が残っています。これはタスク 3-1（sign_in ハンドラーの actorRepo 依存を除去）で対応予定であり、本 PR のスコープ外です。

`sign_up/handler.go` からは `repository` パッケージの import が完全に除去されています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 2-1 の要件がすべて正しく実装されています。

- Validator の `internal/validator/` パッケージへの移動が正確に行われている
- 命名規則（`{Handler}{Action}Validator`）が計画書・ガイドラインと完全に一致している
- Handler の `NewHandler` シグネチャが適切に更新され、`main.go` から Validator を注入する構築パターンに変更されている
- `sign_up/handler.go` から `repository` の import が完全に除去されている
- テストコードも正しく移動・リネームされ、テストの内容自体は変更されていない（リファクタリングのみ）
- 既存の機能に影響を与えない変更である
