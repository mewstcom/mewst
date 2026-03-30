# コードレビュー: sign-up-fix

## レビュー情報

| 項目                       | 内容                          |
| -------------------------- | ----------------------------- |
| レビュー日                 | 2026-02-26                    |
| 対象ブランチ               | sign-up-fix                   |
| ベースブランチ             | go                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md |
| 変更ファイル数             | 60 ファイル（sign-up 関連）   |
| 変更行数（実装）           | +5969 / -141 行               |
| 変更行数（テスト）         | +2708 / -32 行                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/migrations/20260204100000_create_rate_limits.sql`
- [x] `go/db/schema.sql`
- [x] `go/internal/auth/password.go`
- [x] `go/internal/handler/accounts/handler.go`
- [x] `go/internal/handler/accounts/new.go`
- [ ] `go/internal/handler/accounts/create.go`
- [ ] `go/internal/handler/accounts/validator.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/handler/sign_up/new.go`
- [ ] `go/internal/handler/sign_up/create.go`
- [ ] `go/internal/handler/sign_up/validator.go`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/model/profile.go`
- [x] `go/internal/model/user_profile.go`
- [x] `go/internal/query/actors.sql.go`（自動生成）
- [x] `go/internal/query/models.go`（自動生成）
- [x] `go/internal/query/profiles.sql.go`（自動生成）
- [x] `go/internal/query/querier.go`（自動生成）
- [x] `go/internal/query/queries/actors.sql`
- [x] `go/internal/query/queries/profiles.sql`
- [x] `go/internal/query/queries/rate_limits.sql`
- [x] `go/internal/query/queries/user_profiles.sql`
- [x] `go/internal/query/queries/users.sql`
- [x] `go/internal/query/rate_limits.sql.go`（自動生成）
- [x] `go/internal/query/user_profiles.sql.go`（自動生成）
- [x] `go/internal/query/users.sql.go`（自動生成）
- [x] `go/internal/ratelimit/limiter.go`
- [x] `go/internal/repository/actor_repository.go`
- [x] `go/internal/repository/profile_repository.go`
- [ ] `go/internal/repository/rate_limit_repository.go`
- [x] `go/internal/repository/user_profile_repository.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/templates/pages/accounts/new.templ`
- [x] `go/internal/templates/pages/accounts/new_templ.go`（自動生成）
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`（自動生成）
- [x] `go/internal/testutil/db.go`
- [x] `go/internal/usecase/create_account.go`
- [x] `go/internal/usecase/create_email_confirmation.go`
- [x] `go/internal/worker/send_email_confirmation.go`

### テストファイル

- [x] `go/internal/auth/password_test.go`
- [x] `go/internal/handler/accounts/handler_test.go`
- [ ] `go/internal/handler/accounts/validator_test.go`
- [x] `go/internal/handler/email_confirmation/handler_test.go`
- [x] `go/internal/handler/email_confirmation/validator_test.go`
- [x] `go/internal/handler/password_reset/handler_test.go`
- [x] `go/internal/handler/sign_up/handler_test.go`
- [ ] `go/internal/handler/sign_up/validator_test.go`
- [x] `go/internal/ratelimit/limiter_test.go`
- [x] `go/internal/repository/profile_repository_test.go`
- [ ] `go/internal/repository/rate_limit_repository_test.go`
- [x] `go/internal/repository/user_profile_repository_test.go`
- [x] `go/internal/usecase/create_account_test.go`
- [x] `go/internal/usecase/create_email_confirmation_test.go`
- [x] `go/internal/middleware/reverse_proxy_test.go`
- [x] `go/internal/worker/send_email_confirmation_test.go`

### 設定・その他

- [x] `go/go.mod`
- [x] `go/go.sum`

## ファイルごとのレビュー結果

### `go/internal/handler/sign_up/validator.go`: `no-reply` が reservedAtnames にある場合と同様のデッドコードの可能性

**ステータス**: 情報提供

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- **[@go/docs/validation-guide.md]**: `sign_up/validator.go` のバリデーション結果の `FormErrors` に nil チェックがない

  `sign_up/create.go` の 72 行目:

  ```go
  if result.FormErrors.HasErrors() {
  ```

  sign_in の `create.go` では `result.FormErrors != nil &&` を付けて nil チェックを行っている。sign_up のバリデーターは常に `FormErrors` を初期化して返すため実害はないが、防御的プログラミングと sign_in との一貫性の観点から nil チェックを追加すべき。

  **修正案**:

  ```go
  if result.FormErrors != nil && result.FormErrors.HasErrors() {
  ```

  **対応方針**:
  - [x] 修正案の通り nil チェックを追加する
  - [ ] 現状のまま（実害がないため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/sign_up/create.go`: レート制限のテストが不足

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- **[@go/CLAUDE.md#テスト戦略]**: `sign_up/create.go` の 39-50 行目にレート制限処理があるが、対応するテスト（`TestCreate_RateLimitExceeded` 等）が `handler_test.go` に存在しない

  レート制限はセキュリティ上重要な機能であり、テストで動作を検証すべき。`accounts/handler_test.go` にも同様にレート制限のテストが不足している。

  **修正案**:

  `handler_test.go` にレート制限超過時のテストケースを追加する。

  **対応方針**:
  - [x] sign_up と accounts の両方にレート制限テストを追加する
  - [ ] 今回は対応せず、別タスクで対応する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/sign_up/validator_test.go`: テストパッケージ名の不一致

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- **[@go/CLAUDE.md#テスト戦略]**: `sign_up/validator_test.go` は `package sign_up_test`（外部テストパッケージ）を使用しているが、`sign_in/validator_test.go` は `package sign_in`（内部テストパッケージ）を使用している

  同一リポジトリ内で validator テストのパッケージ命名が統一されていない。外部テストパッケージを使用すると import エイリアスが必要になり、コードが冗長になる。

  **修正案**:

  `sign_up/validator_test.go` を `package sign_up` に変更し、sign_in と一致させる。`accounts/validator_test.go` も同様に確認する。

  **対応方針**:
  - [x] sign_up と accounts の validator_test.go を内部テストパッケージに変更する
  - [ ] sign_in の validator_test.go を外部テストパッケージに変更して統一する
  - [ ] 現状のまま（どちらも有効なパターンのため）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/accounts/validator.go`: `no-reply` が予約アットネームに含まれているがマッチしない

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

**問題点・改善提案**:

- **[@go/docs/validation-guide.md]**: `accounts/validator.go` の 27 行目に `"no-reply": true` が `reservedAtnames` マップに含まれているが、14 行目の正規表現 `^[A-Za-z0-9_]+$` はハイフンを許可しない

  つまり `no-reply` はフォーマットチェック（118 行目）で先に拒否されるため、予約名チェック（120 行目）に到達しない。デッドコードとなっている。

  **修正案**:

  `reservedAtnames` マップから `"no-reply"` を削除する。設計書（sign-up.md 416 行目）にも `"no-reply"` が記載されているが、正規表現との整合性を取るべき。

  **対応方針**:
  - [x] `"no-reply"` を `reservedAtnames` から削除する
  - [ ] 防御的に残しておく（将来的に正規表現が変わる可能性を考慮）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/handler/accounts/create.go`: パスワード最小長チェックがバイト数ベース

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

**問題点・改善提案**:

- **[@go/docs/validation-guide.md]**: `accounts/validator.go` の 133 行目で `len(password) < 8` を使用しているが、`len()` はバイト数を返す

  日本語など多バイト文字のパスワードの場合、3 文字（9 バイト）でも最小長チェックを通過してしまう。設計書では「最小長: 8 文字」と記載されている。

  一方、最大長チェック `len(password) > 72` はバイト数が正しい（bcrypt の 72 バイト制限）。

  **修正案**:

  最小長チェックのみ `utf8.RuneCountInString(password) < 8` に変更する。最大長は `len(password) > 72` のまま維持。

  ```go
  import "unicode/utf8"

  if utf8.RuneCountInString(password) < 8 {
      formErrors.AddFieldError("password", templates.T(ctx, "error_password_too_short"))
  }
  ```

  **対応方針**:
  - [x] 最小長チェックを `utf8.RuneCountInString` に変更する
  - [ ] 現状のまま（バイト数ベースで統一、多くの実装で一般的）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/repository/rate_limit_repository_test.go`: テストパッケージ名の不一致

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- **[@go/CLAUDE.md#テスト戦略]**: `rate_limit_repository_test.go` は `package repository`（内部テストパッケージ）を使用しているが、同じディレクトリの `profile_repository_test.go` と `user_profile_repository_test.go` は `package repository_test`（外部テストパッケージ）を使用している

  Repository テストは公開 API をテストすべきであり、外部テストパッケージの使用が推奨される。

  **修正案**:

  `rate_limit_repository_test.go` のパッケージ名を `package repository_test` に変更する。

  **対応方針**:
  - [x] `package repository_test` に変更する
  - [ ] 現状のまま
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

全体的に設計書に忠実に実装されており、3 層アーキテクチャ、WithTx パターン、ハンドラーガイドラインに従った実装がなされています。

## 総合評価

**評価**: Comment

**総評**:

サインアップ機能の実装は全体的に高品質で、設計書の全要件を網羅しています。

**良かった点**:

- 3 層アーキテクチャの依存関係ルールが正確に守られている
- セキュリティ対策（CSRF、Turnstile、レート制限、bcrypt、入力バリデーション）が包括的に実装されている
- I18n が完全に対応されており、ja.toml と en.toml が対称的に定義されている
- テンプレートが sign_in の既存パターンと完全に一致している
- テストカバレッジが高く、正常系・異常系を網羅している
- WithTx パターンによるトランザクション管理が正確に実装されている

**指摘事項のサマリー**:

- 必須対応: 0 件
- 推奨対応: 3 件（テストパッケージ名の不一致 x2、レート制限テストの不足）
- 要確認: 3 件（FormErrors nil チェック、`no-reply` デッドコード、パスワード最小長のバイト/文字数）
- 設計との乖離: 0 件
