# コードレビュー: fix-validation

## レビュー情報

| 項目             | 内容                           |
| ---------------- | ------------------------------ |
| レビュー日       | 2026-02-03                     |
| 対象ブランチ     | fix-validation                 |
| ベースブランチ   | go                             |
| 変更ファイル数   | 87 ファイル                    |
| 変更行数（実装） | +3383 / -964 行                |
| 変更行数（テスト）| 上記に含まれる                 |

## 参照するガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - プロジェクト全体のガイドライン
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in/new.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/validator.go`
- [x] `go/internal/handler/sign_in/request.go`
- [x] `go/internal/handler/password_reset/handler.go`
- [x] `go/internal/handler/password_reset/new.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/password_reset/validator.go`
- [x] `go/internal/handler/password_reset/request.go`
- [x] `go/internal/handler/password/handler.go`
- [x] `go/internal/handler/password/edit.go`
- [x] `go/internal/handler/password/update.go`
- [x] `go/internal/handler/password/validator.go`
- [x] `go/internal/handler/email_confirmation/handler.go`
- [x] `go/internal/handler/email_confirmation/new.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/validator.go`
- [x] `go/internal/handler/email_confirmation/request.go`
- [x] `go/internal/handler/sign_out/handler.go`
- [x] `go/internal/handler/sign_out/delete.go`
- [x] `go/internal/handler/manifest/handler.go`
- [x] `go/internal/handler/manifest/show.go`
- [x] `go/internal/handler/manifest/show_test.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/repository/session_repository.go`
- [x] `go/internal/repository/email_confirmation_repository.go`
- [x] `go/internal/repository/actor_repository.go`
- [x] `go/internal/usecase/create_session.go`
- [x] `go/internal/usecase/create_email_confirmation.go`
- [x] `go/internal/usecase/update_password.go`
- [x] `go/internal/usecase/mark_email_as_confirmed.go`
- [x] `go/internal/usecase/confirm_email.go`
- [x] `go/internal/middleware/auth.go`
- [x] `go/internal/middleware/csrf.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/session/manager.go`
- [x] `go/internal/viewmodel/page_meta.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_email.go`

### テストファイル

- [x] `go/internal/handler/sign_in/handler_test.go`
- [x] `go/internal/handler/sign_in/validator_test.go`
- [x] `go/internal/handler/password_reset/handler_test.go`
- [x] `go/internal/handler/password_reset/validator_test.go`
- [x] `go/internal/handler/password/handler_test.go`
- [x] `go/internal/handler/password/validator_test.go`
- [x] `go/internal/handler/email_confirmation/handler_test.go`
- [x] `go/internal/handler/email_confirmation/validator_test.go`
- [x] `go/internal/handler/sign_out/handler_test.go`
- [x] `go/internal/repository/user_repository_test.go`
- [x] `go/internal/repository/session_repository_test.go`
- [x] `go/internal/repository/email_confirmation_repository_test.go`
- [x] `go/internal/repository/actor_repository_test.go`
- [x] `go/internal/usecase/create_session_test.go`
- [x] `go/internal/usecase/create_email_confirmation_test.go`
- [x] `go/internal/usecase/update_password_test.go`
- [x] `go/internal/usecase/mark_email_as_confirmed_test.go`
- [x] `go/internal/usecase/confirm_email_test.go`
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
- [x] `docs/reviews/template.md`
- [x] `rails/CLAUDE.md`

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

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- **[@go/docs/validation-guide.md#命名規則]**: バリデーターの構造が統一された命名パターンに従っていない

  ガイドラインでは `{Action}Validator` 構造体と `{Action}ValidatorInput`、`{Action}ValidatorResult` を使用し、`Validate(ctx, input)` メソッドを持つパターンを推奨しています。

  現在の実装:
  ```go
  type CreateValidator struct {
      Email string  // 入力値を構造体に直接持っている
  }

  func (v *CreateValidator) Validate(ctx context.Context) *session.FormErrors {
      // Inputを受け取らず、自身のフィールドを使用
  }
  ```

  ガイドラインに従った形式:
  ```go
  type CreateValidator struct{}

  type CreateValidatorInput struct {
      Email string
  }

  type CreateValidatorResult struct {
      FormErrors *session.FormErrors
  }

  func (v *CreateValidator) Validate(ctx context.Context, input CreateValidatorInput) *CreateValidatorResult {
      // ...
  }
  ```

  **推奨**: 他のハンドラー（`sign_in`、`email_confirmation`）では正しいパターンが使われているため、`password_reset` も統一することを検討してください。

### `go/internal/handler/password/validator.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- **[@go/docs/validation-guide.md#命名規則]**: 上記 `password_reset/validator.go` と同様の問題

  現在の実装:
  ```go
  type UpdateValidator struct {
      Password string  // 入力値を構造体に直接持っている
  }

  func (v *UpdateValidator) Validate(ctx context.Context) *session.FormErrors {
      // Inputを受け取らず、自身のフィールドを使用
  }
  ```

  **推奨**: 統一されたパターンへの変更を検討してください。

### `go/internal/handler/sign_in/validator.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- 問題なし。ガイドラインに従った構造で実装されています。

### `go/internal/handler/email_confirmation/validator.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- 問題なし。ガイドラインに従った構造で実装されています。

### `go/internal/handler/sign_in/handler.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 依存性注入のガイドライン

**問題点・改善提案**:

- 問題なし。Handler 構造体の定義と NewHandler コンストラクタが適切に実装されています。

### `go/internal/handler/password_reset/handler.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし。

### `go/internal/handler/password/handler.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし。

### `go/internal/handler/email_confirmation/handler.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし。バリデーターがコンストラクタで初期化されている点が良い。

### `go/internal/handler/sign_in/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力ガイドライン

**問題点・改善提案**:

- 問題なし。`slog` を適切に使用し、エラーハンドリングも適切です。

### `go/internal/handler/password_reset/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド
- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- 良い点: メール確認レコード作成に失敗しても、セキュリティ上の理由でユーザーには成功メッセージを表示している（メールアドレスの存在確認を防ぐため）。

### `go/internal/handler/password/update.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし。パスワード更新後のセッション無効化処理が適切に実装されています。

### `go/internal/handler/email_confirmation/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし。イベントに応じたリダイレクト処理が適切に分離されています。

### `go/internal/repository/user_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし。`WithTx` パターンが適切に実装されています。

### `go/internal/repository/session_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし。

### `go/internal/repository/email_confirmation_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし。`toModel` ヘルパーメソッドによる変換が適切に実装されています。

### `go/internal/repository/actor_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし。

### `go/internal/usecase/create_session.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/CLAUDE.md#ユースケース](/workspace/go/CLAUDE.md) - ユースケースのガイドライン

**問題点・改善提案**:

- 問題なし。命名規則と Execute メソッドが適切です。

### `go/internal/usecase/create_email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし。非同期メール送信と同期メール送信の両方に対応した設計が良い。

### `go/internal/usecase/update_password.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし。

### `go/internal/usecase/mark_email_as_confirmed.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- 問題なし。

### `go/internal/middleware/auth.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- 問題なし。`RequireAuth`、`RequireNoAuth`、`SetUser` の3つのミドルウェアが適切に分離されています。

### `go/internal/middleware/csrf.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - セキュリティガイドライン

**問題点・改善提案**:

- 問題なし。CSRF保護が適切に実装されています。

### `go/internal/middleware/reverse_proxy.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#リバースプロキシによる段階的移行](/workspace/go/CLAUDE.md) - リバースプロキシの説明

**問題点・改善提案**:

- 問題なし。ホワイトリスト方式で Go 版で処理するパスが適切に定義されています。

### `go/internal/session/manager.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版開発ガイド

**問題点・改善提案**:

- 問題なし。Rails 版との互換性を保ちながら適切に実装されています。

### `go/internal/handler/manifest/handler.go` と `show.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし。単独エンドポイントでもディレクトリ化されている点が良い。

### `go/internal/handler/sign_out/handler.go` と `delete.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - ハンドラーガイド

**問題点・改善提案**:

- 問題なし。

## 総合評価

**評価**: Approve

**総評**:

全体的に良く設計された変更です。3層アーキテクチャの原則に従い、Handler、UseCase、Repository の責務が明確に分離されています。

**良かった点**:

1. **バリデーションの統合**: `sign_in` と `email_confirmation` では、形式バリデーションと状態バリデーションが1つの `validator.go` に適切に統合されています
2. **セキュリティ考慮**: パスワードリセット時にメールアドレスの存在を漏らさない設計
3. **一貫した命名規則**: ファイル名とメソッド名の対応が明確
4. **適切なエラーハンドリング**: `slog` を使用した構造化ログ
5. **テストの充実**: 各コンポーネントに対応するテストファイルが追加されている

**修正済みの問題点**:

1. ~~`password_reset/validator.go` と `password/validator.go` のバリデーター構造が他のハンドラーと異なるパターンを使用している~~ → **修正完了**（2026-02-03）

---

## 質問と回答

### Q1: バリデーターの構造統一について

**種別**: 推奨

**背景**:

`sign_in/validator.go` と `email_confirmation/validator.go` では、以下のパターンが使われています:
- `{Action}Validator` 構造体（依存性のみを持つ）
- `{Action}ValidatorInput` 構造体（入力データ）
- `{Action}ValidatorResult` 構造体（結果）
- `Validate(ctx, input) *Result` メソッド

一方、`password_reset/validator.go` と `password/validator.go` では:
- `{Action}Validator` 構造体（入力データを直接持つ）
- `Validate(ctx) *session.FormErrors` メソッド

この違いは意図的なものでしょうか？

**選択肢**:

- [ ] 選択肢A: 意図的である。形式バリデーションのみの場合はシンプルなパターンを使う
- [ ] 選択肢B: 統一すべき。すべてのバリデーターを同じパターンに変更する
- [x] その他（下の回答欄に記入）

**回答**:

```
意図していませんでした。
`password_reset/validator.go` と `password/validator.go` もガイドラインのように実装されているべきです。
```
