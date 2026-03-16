# コードレビュー: sign-up-fix

## レビュー情報

| 項目                       | 内容                                     |
| -------------------------- | ---------------------------------------- |
| レビュー日                 | 2026-02-26                               |
| 対象ブランチ               | sign-up-fix                              |
| ベースブランチ             | go                                       |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md            |
| 変更ファイル数             | 71 ファイル（Go 関連）                   |
| 変更行数（実装）           | 約 +6268 / -173 行（テスト除く）         |
| 変更行数（テスト）         | 約 +2821 / -32 行                        |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/auth/password.go`
- [x] `go/internal/handler/accounts/create.go`
- [x] `go/internal/handler/accounts/handler.go`
- [x] `go/internal/handler/accounts/new.go`
- [x] `go/internal/handler/accounts/validator.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/sign_up/create.go`
- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/handler/sign_up/new.go`
- [x] `go/internal/handler/sign_up/validator.go`
- [x] `go/internal/middleware/reverse_proxy.go`
- [x] `go/internal/model/profile.go`
- [x] `go/internal/model/user_profile.go`
- [x] `go/internal/query/actors.sql.go`
- [x] `go/internal/query/models.go`
- [x] `go/internal/query/profiles.sql.go`
- [x] `go/internal/query/querier.go`
- [x] `go/internal/query/queries/actors.sql`
- [x] `go/internal/query/queries/profiles.sql`
- [x] `go/internal/query/queries/rate_limits.sql`
- [x] `go/internal/query/queries/user_profiles.sql`
- [x] `go/internal/query/queries/users.sql`
- [x] `go/internal/query/rate_limits.sql.go`
- [x] `go/internal/query/user_profiles.sql.go`
- [x] `go/internal/query/users.sql.go`
- [x] `go/internal/ratelimit/limiter.go`
- [x] `go/internal/repository/actor_repository.go`
- [x] `go/internal/repository/profile_repository.go`
- [x] `go/internal/repository/rate_limit_repository.go`
- [x] `go/internal/repository/user_profile_repository.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/templates/pages/accounts/new.templ`
- [x] `go/internal/templates/pages/accounts/new_templ.go`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`
- [x] `go/internal/testutil/db.go`
- [x] `go/internal/usecase/create_account.go`
- [x] `go/internal/usecase/create_email_confirmation.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_email_confirmation.go`

### テストファイル

- [x] `go/internal/auth/password_test.go`
- [x] `go/internal/handler/accounts/handler_test.go`
- [x] `go/internal/handler/accounts/validator_test.go`
- [x] `go/internal/handler/email_confirmation/handler_test.go`
- [x] `go/internal/handler/email_confirmation/validator_test.go`
- [x] `go/internal/handler/password_reset/handler_test.go`
- [x] `go/internal/handler/sign_up/handler_test.go`
- [x] `go/internal/handler/sign_up/validator_test.go`
- [x] `go/internal/middleware/reverse_proxy_test.go`
- [x] `go/internal/ratelimit/limiter_test.go`
- [x] `go/internal/repository/profile_repository_test.go`
- [x] `go/internal/repository/rate_limit_repository_test.go`
- [x] `go/internal/repository/user_profile_repository_test.go`
- [x] `go/internal/usecase/create_account_test.go`
- [x] `go/internal/usecase/create_email_confirmation_test.go`
- [x] `go/internal/worker/send_email_confirmation_test.go`

### 設定・その他

- [x] `go/.air.toml`
- [x] `go/.golangci.yml`
- [x] `go/db/migrations/20260204100000_create_rate_limits.sql`
- [x] `go/db/migrations/20260226100000_drop_redundant_rate_limits_index.sql`
- [x] `go/db/schema.sql`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/mise.toml`
- [x] `mise.toml`
- [x] `docker-compose.yml`
- [x] `.github/workflows/fmt-ci.yml`
- [x] `.github/workflows/go-ci.yml`
- [x] `.github/workflows/rails-ci.yml`
- [x] `rails/mise.toml`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに従っています。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

サインアップ機能の Go 版移植が作業計画書に沿って適切に実装されています。全6フェーズ（基盤整備、サインアップフォーム、メール確認フロー拡張、アカウント作成フォーム、統合とルーティング、Rails版セッション対応）がすべて完了しており、設計書との乖離はありません。

**良かった点**:

- **アーキテクチャの一貫性**: 3層アーキテクチャ（Presentation → Application → Domain/Infrastructure）の依存関係ルールが正しく守られている。Handler → UseCase/Repository → Query の依存方向が適切
- **セキュリティ対策の充実**: CSRF トークン、Cloudflare Turnstile、bcrypt パスワードハッシュ化、PostgreSQL ベースのレート制限がすべて正しく実装されている
- **ハンドラーガイドラインの遵守**: 標準ファイル名（handler.go, new.go, create.go, validator.go）を使用し、1エンドポイント=1ファイルの原則に従っている
- **バリデーション設計**: 形式バリデーションと状態バリデーション（DB検証）が validator.go に統合されており、ガイドラインに準拠。メールアドレスの重複はグローバルエラー、アットネームの重複はフィールドエラーとして適切に使い分けている
- **テンプレートの構造体パターン**: `NewPageData` 構造体ベースの引数パターンを正しく使用。context.Context を明示的に渡していない
- **国際化の完全対応**: すべてのユーザー向けメッセージが ja.toml / en.toml の両方に追加されている
- **UseCase の WithTx パターン**: `CreateAccountUsecase` でのトランザクション管理が正しく実装されている（Profile, User, UserProfile, Actor の4エンティティを1トランザクションで作成）
- **テストの充実**: ハンドラーテスト、バリデーションテスト、ユースケーステスト、リポジトリテストが網羅的に書かれている。既存テストパターン（`testutil.SetupTestDB(t)`）との一貫性も保たれている
- **既存コードとの一貫性**: 既存の sign_in, password_reset, email_confirmation ハンドラーと同じパターンで実装されている
