# コードレビュー: sign-up-fix

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-02-26                               |
| 対象ブランチ               | sign-up-fix                              |
| ベースブランチ             | go                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md            |
| 変更ファイル数             | 54 ファイル（sign-up 関連の Go コード）  |
| 変更行数（実装）           | +2600 行程度                             |
| 変更行数（テスト）         | +2800 行程度                             |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/handler/sign_up/new.go`
- [x] `go/internal/handler/sign_up/create.go`
- [x] `go/internal/handler/sign_up/validator.go`
- [x] `go/internal/handler/accounts/handler.go`
- [x] `go/internal/handler/accounts/new.go`
- [x] `go/internal/handler/accounts/create.go`
- [x] `go/internal/handler/accounts/validator.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/usecase/create_account.go`
- [x] `go/internal/usecase/create_email_confirmation.go`
- [x] `go/internal/repository/profile_repository.go`
- [x] `go/internal/repository/user_profile_repository.go`
- [x] `go/internal/repository/actor_repository.go`
- [x] `go/internal/repository/rate_limit_repository.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/model/profile.go`
- [x] `go/internal/model/user_profile.go`
- [x] `go/internal/ratelimit/limiter.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/auth/password.go`
- [x] `go/internal/worker/send_email_confirmation.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/accounts/new.templ`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/testutil/db.go`
- [x] `go/internal/query/queries/actors.sql`
- [x] `go/internal/query/queries/profiles.sql`
- [x] `go/internal/query/queries/rate_limits.sql`
- [x] `go/internal/query/queries/user_profiles.sql`
- [x] `go/internal/query/queries/users.sql`
- [x] `go/db/migrations/20260204100000_create_rate_limits.sql`
- [x] `go/db/migrations/20260226100000_drop_redundant_rate_limits_index.sql`

### テストファイル

- [x] `go/internal/handler/sign_up/handler_test.go`
- [x] `go/internal/handler/sign_up/validator_test.go`
- [x] `go/internal/handler/accounts/handler_test.go`
- [x] `go/internal/handler/accounts/validator_test.go`
- [x] `go/internal/handler/email_confirmation/handler_test.go`
- [x] `go/internal/handler/email_confirmation/validator_test.go`
- [x] `go/internal/handler/password_reset/handler_test.go`
- [x] `go/internal/usecase/create_account_test.go`
- [x] `go/internal/usecase/create_email_confirmation_test.go`
- [x] `go/internal/repository/profile_repository_test.go`
- [x] `go/internal/repository/user_profile_repository_test.go`
- [x] `go/internal/repository/rate_limit_repository_test.go`
- [x] `go/internal/ratelimit/limiter_test.go`
- [x] `go/internal/middleware/reverse_proxy_test.go`
- [x] `go/internal/auth/password_test.go`
- [x] `go/internal/worker/send_email_confirmation_test.go`

### 設定・その他

- [x] `go/db/schema.sql`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] sqlc 自動生成ファイル群（`go/internal/query/*.go`）

## ファイルごとのレビュー結果

### `go/internal/handler/sign_up/validator_test.go`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テーブル駆動テストの書き方](/workspace/go/CLAUDE.md)

**問題点・改善提案**:

- **[@go/CLAUDE.md#テーブル駆動テストの書き方]**: `TestCreateValidator_InvalidEmailFormat` と `TestCreateValidator_ValidEmail` で `for` ループでテストケースを回しているが、`t.Run` を使っていない

  テストガイドラインでは、複数のテストケースを回す場合は `t.Run` でサブテスト化してテスト失敗時にどのケースが原因か特定しやすくすることが推奨されている。

  ```go
  // 現在のコード (validator_test.go)
  for _, email := range invalidEmails {
      result := v.Validate(ctx, CreateValidatorInput{Email: email})
      if !result.FormErrors.HasFieldError("email") {
          t.Errorf(...)
      }
  }
  ```

  **修正案**:

  ```go
  for _, email := range invalidEmails {
      t.Run(email, func(t *testing.T) {
          result := v.Validate(ctx, CreateValidatorInput{Email: email})
          if !result.FormErrors.HasFieldError("email") {
              t.Errorf(...)
          }
      })
  }
  ```

  **対応方針**:

  - [x] 修正案の通り `t.Run` を使うように変更する
  - [ ] 現状のまま（テストは動作しており、エラーメッセージに email 値が含まれている）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

サインアップ機能の実装は、作業計画書に記載された全フェーズ（基盤整備、サインアップフォーム、メール確認フロー拡張、アカウント作成フォーム、統合とルーティング）が正しく実装されており、品質の高いコードです。

**良い点**:

- **アーキテクチャ遵守**: 3 層アーキテクチャのルールが完全に守られている。Handler/UseCase が Query に直接依存することなく、全てのデータアクセスが Repository 経由
- **セキュリティ**: CSRF トークン、Turnstile Bot 対策、レート制限、bcrypt パスワードハッシュ化が全て適切に実装。機密データがログに出力されない
- **命名規則**: ファイル名（9 種類の標準ファイル名）、メソッド名、構造体名が全てガイドラインに一致
- **国際化**: 全てのユーザー向けメッセージが `templates.T(ctx, ...)` で国際化されている
- **テスト**: 正常系・異常系の網羅的なテスト。実 DB を使ったトランザクション分離テスト、`t.Parallel()` による並行実行
- **既存コードとの一貫性**: sign_in ハンドラーのパターンを踏襲しており、コードベース全体の一貫性が保たれている
- **設計書との整合性**: 作業計画書のサインアップフロー（メールアドレス入力 → 確認コード送信 → コード検証 → アカウント詳細入力 → アカウント作成）が全て正しく実装

**軽微な観察事項（修正不要）**:

- ハンドラー内のロケールが `"ja"` にハードコードされている（`ctx = templates.WithLocale(ctx, "ja")`）。これは sign_in 等の既存ハンドラーと一致するパターンであり、多言語対応のミドルウェア実装時にまとめて対応すべき事項
- `accounts/handler.go` の Handler 構造体フィールドが 8 個（ガイドラインの上限ちょうど）。NewHandler の引数は 9 個だが、うち 2 個（userRepo, profileRepo）は validator の内部初期化にのみ使用されており、Handler 構造体には含まれないため問題なし
- `http.StatusFound` (302) を POST 後のリダイレクトで使用（`http.StatusSeeOther` (303) の方がセマンティクス上は正確だが、既存パターンと一致）
