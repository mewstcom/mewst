# コードレビュー: fix-validation（フォローアップ）

## レビュー情報

| 項目              | 内容                      |
| ----------------- | ------------------------- |
| レビュー日        | 2026-02-03                |
| 対象ブランチ      | fix-validation            |
| ベースブランチ    | go                        |
| 変更ファイル数    | 89 ファイル               |
| 変更行数（実装）  | +3992 / -1000 行          |
| 変更行数（テスト）| 上記に含まれる            |

## 参照するガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - プロジェクト全体のガイドライン
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル（バリデーター関連）

- [x] `go/internal/handler/password_reset/validator.go`
- [x] `go/internal/handler/password/validator.go`
- [x] `go/internal/handler/sign_in/validator.go`
- [x] `go/internal/handler/email_confirmation/validator.go`

### 実装ファイル（ハンドラー関連）

- [x] `go/internal/handler/password_reset/handler.go`
- [x] `go/internal/handler/password_reset/new.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/password/handler.go`
- [x] `go/internal/handler/password/edit.go`
- [x] `go/internal/handler/password/update.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in/new.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/email_confirmation/handler.go`
- [x] `go/internal/handler/email_confirmation/new.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/sign_out/handler.go`
- [x] `go/internal/handler/sign_out/delete.go`
- [x] `go/internal/handler/manifest/handler.go`
- [x] `go/internal/handler/manifest/show.go`

### 実装ファイル（ユースケース・リポジトリ）

- [x] `go/internal/usecase/create_session.go`
- [x] `go/internal/usecase/create_email_confirmation.go`
- [x] `go/internal/usecase/update_password.go`
- [x] `go/internal/usecase/mark_email_as_confirmed.go`
- [x] `go/internal/usecase/confirm_email.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/repository/session_repository.go`
- [x] `go/internal/repository/email_confirmation_repository.go`
- [x] `go/internal/repository/actor_repository.go`

### 実装ファイル（ミドルウェア・セッション・その他）

- [x] `go/internal/middleware/auth.go`
- [x] `go/internal/middleware/csrf.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/session/manager.go`
- [x] `go/internal/viewmodel/page_meta.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_email.go`

### テストファイル

- [x] `go/internal/handler/password_reset/validator_test.go`
- [x] `go/internal/handler/password_reset/handler_test.go`
- [x] `go/internal/handler/password/validator_test.go`
- [x] `go/internal/handler/password/handler_test.go`
- [x] `go/internal/handler/sign_in/validator_test.go`
- [x] `go/internal/handler/sign_in/handler_test.go`
- [x] `go/internal/handler/email_confirmation/validator_test.go`
- [x] `go/internal/handler/email_confirmation/handler_test.go`
- [x] `go/internal/handler/sign_out/handler_test.go`
- [x] `go/internal/handler/manifest/show_test.go`
- [x] `go/internal/usecase/create_session_test.go`
- [x] `go/internal/usecase/create_email_confirmation_test.go`
- [x] `go/internal/usecase/update_password_test.go`
- [x] `go/internal/usecase/mark_email_as_confirmed_test.go`
- [x] `go/internal/usecase/confirm_email_test.go`
- [x] `go/internal/repository/user_repository_test.go`
- [x] `go/internal/repository/session_repository_test.go`
- [x] `go/internal/repository/email_confirmation_repository_test.go`
- [x] `go/internal/repository/actor_repository_test.go`
- [x] `go/internal/middleware/auth_test.go`
- [x] `go/internal/middleware/csrf_test.go`
- [x] `go/internal/middleware/reverse_proxy_test.go`
- [x] `go/internal/session/manager_test.go`
- [x] `go/internal/session/flash_test.go`
- [x] `go/internal/worker/send_email_test.go`
- [x] `go/internal/email/sender_test.go`
- [x] `go/internal/clientip/clientip_test.go`

### 設定・ドキュメント

- [x] `go/CLAUDE.md`
- [x] `go/.golangci.yml`
- [x] `go/go.mod`
- [x] `go/cmd/server/main.go`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/handler-guide.md`
- [x] `go/docs/validation-guide.md`
- [x] `CLAUDE.md`
- [x] `rails/CLAUDE.md`
- [x] `docs/reviews/template.md`

### テンプレートファイル

- [x] `go/internal/templates/components/flash_templ.go`
- [x] `go/internal/templates/components/form_errors_templ.go`
- [x] `go/internal/templates/components/head_templ.go`
- [x] `go/internal/templates/helper.go`
- [x] `go/internal/templates/layouts/simple_templ.go`
- [x] `go/internal/templates/pages/email_confirmation/new_templ.go`
- [x] `go/internal/templates/pages/password/edit_templ.go`
- [x] `go/internal/templates/pages/password_reset/new_templ.go`
- [x] `go/internal/templates/pages/sign_in/new_templ.go`

## ファイルごとのレビュー結果

### `go/internal/handler/password_reset/validator.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- **修正済み**: 以前の問題（バリデーターの構造がガイドラインと異なる）は解消されています
- 現在の実装は `CreateValidator` + `CreateValidatorInput` + `CreateValidatorResult` のパターンに従っています
- `net/mail.ParseAddress` を使ったメール形式チェックは適切です

### `go/internal/handler/password/validator.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- **修正済み**: ガイドラインに従った構造に変更されています
- パスワードの文字数チェック（8文字以上）とバイト数チェック（72バイト以下、bcrypt制限）が適切に実装されています
- `len([]rune(input.Password))` で文字数をカウントしている点が良い（日本語対応）

### `go/internal/handler/sign_in/validator.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- 問題なし
- 形式バリデーション → 状態バリデーション（DB検証）の順序が適切
- `ErrNotFound` のハンドリングで `AddGlobalError` を使用し、メールアドレスの存在を漏らさない設計

### `go/internal/handler/email_confirmation/validator.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- 問題なし
- 確認コードの形式（6桁数字）を正規表現でチェックしている点が良い
- コード不一致時のエラーメッセージ（`error_code_incorrect_or_expired`）でコードの有効期限も示唆している点がセキュリティ上適切

### `go/internal/handler/password_reset/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- 問題なし
- Turnstile検証 → バリデーション → ユースケース実行の順序が適切
- メール確認作成失敗時も成功レスポンスを返す設計がセキュリティ上良い

### `go/internal/handler/password/update.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし
- `GetSucceededByID` で確認済みレコードのみを取得している点が適切
- パスワード更新後にセッションを無効化している点がセキュリティ上良い

### `go/internal/handler/sign_in/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- 問題なし
- オープンリダイレクト攻撃を防ぐためのバリデーション（`strings.HasPrefix(backURL, "/") && !strings.HasPrefix(backURL, "//")`）が適切

### `go/internal/handler/email_confirmation/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし
- イベント種別に応じたリダイレクト先の分岐（`getRedirectPath`）が適切に分離されている

### `go/internal/usecase/update_password.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし
- シンプルな単一責任のユースケース

### `go/internal/usecase/create_email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし
- `WithWorkerClient` パターンで同期/非同期メール送信を切り替えられる設計が良い
- 確認コードの生成（6桁ゼロ埋め）が適切

### `go/internal/usecase/mark_email_as_confirmed.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし

### `go/internal/repository/email_confirmation_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし
- `GetActiveByID`（有効期限内かつ未確認）と `GetSucceededByID`（確認済み）の分離が適切
- `toModel` ヘルパーによる型変換が一貫している

### `go/internal/handler/password_reset/validator_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし
- テーブル駆動テストで多様なメールアドレスパターンをカバー
- 正常系と異常系が適切に分類されている

### `go/internal/handler/password/validator_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし
- 境界値テスト（7文字/8文字、72バイト/73バイト）が適切に実装されている
- 日本語パスワードのテストケースも含まれている点が良い

## 総合評価

**評価**: Approve

**総評**:

前回のレビュー（`20260203-fix-validation.md`）で指摘された問題点（`password_reset/validator.go` と `password/validator.go` のバリデーター構造の不統一）が適切に修正されています。

**良かった点**:

1. **バリデーターの統一**: すべてのバリデーターが `{Action}Validator` + `{Action}ValidatorInput` + `{Action}ValidatorResult` のパターンに統一されました
2. **セキュリティ考慮**:
   - メールアドレスの存在を漏らさない設計
   - オープンリダイレクト攻撃の防止
   - パスワード更新後のセッション無効化
3. **テストの充実**: 境界値テスト、日本語対応テストなど、エッジケースも網羅
4. **一貫したエラーハンドリング**: `slog` を使用した構造化ログ、適切なHTTPステータスコード

**問題点**:

- なし

---

## 質問と回答

（質問なし）
