# コードレビュー: sign-up ブランチ

## レビュー情報

| 項目                       | 内容                                  |
| -------------------------- | ------------------------------------- |
| レビュー日                 | 2026-02-25                            |
| 対象ブランチ               | sign-up                               |
| ベースブランチ             | go                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md         |
| 変更ファイル数             | 126 ファイル（Go 実装 31, テスト 16） |
| 変更行数（実装）           | +2066 / -100 行                       |
| 変更行数（テスト）         | +2204 / -30 行                        |

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
- [x] `go/internal/query/rate_limits.sql.go`
- [x] `go/internal/query/user_profiles.sql.go`
- [x] `go/internal/query/users.sql.go`
- [x] `go/internal/ratelimit/limiter.go`
- [x] `go/internal/repository/actor_repository.go`
- [x] `go/internal/repository/profile_repository.go`
- [x] `go/internal/repository/rate_limit_repository.go`
- [x] `go/internal/repository/user_profile_repository.go`
- [x] `go/internal/repository/user_repository.go`
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

### テンプレートファイル

- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/accounts/new.templ`

### 設定・その他

- [x] `go/db/migrations/20260204100000_create_rate_limits.sql`
- [x] `go/db/schema.sql`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/query/queries/actors.sql`
- [x] `go/internal/query/queries/profiles.sql`
- [x] `go/internal/query/queries/rate_limits.sql`
- [x] `go/internal/query/queries/user_profiles.sql`
- [x] `go/internal/query/queries/users.sql`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] 他多数（CI設定、ドキュメント等）

## ファイルごとのレビュー結果

### `go/internal/handler/accounts/handler.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md#依存性注入のガイドライン](/workspace/go/docs/handler-guide.md) - 依存性注入

**問題点・改善提案**:

- **[@go/docs/handler-guide.md#肥大化の防止]**: Handler 構造体のフィールドが 8 個で、ガイドラインの上限に達している。現状は上限ぴったりなので問題はないが、`NewHandler` のコンストラクタ引数が 9 個（`userRepo` と `profileRepo` は validator 初期化用に追加で渡されている）あり、やや多い。今後の機能追加で超える可能性がある。

  ```go
  // 現状: Handler構造体は8フィールド（上限ぴったり）
  type Handler struct {
      cfg                   *config.Config
      sessionMgr            *session.Manager
      emailConfirmationRepo *repository.EmailConfirmationRepository
      createAccountUC       *usecase.CreateAccountUsecase
      createSessionUC       *usecase.CreateSessionUsecase
      turnstile             turnstile.Verifier
      rateLimiter           *ratelimit.Limiter
      validator             *CreateValidator
  }
  ```

  **修正案**:

  現時点では修正不要。ただし、今後フィールドが増える場合はリソース分割を検討すること。

  **対応方針**:
  - [ ] 現状のまま（フィールド追加時にリソース分割を検討）
  - [ ] 今の段階でリソースを分割する
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  とりあえず現状のままで大丈夫です。解決方法としてはリソースの分割しか無いでしょうか？
  リソースの分割というのは具体的にはどういうことをしますか？
  ```

  **レビュアー回答**:

  はい、基本的にはリソースの分割が主な解決方法です。具体的には、handler-guide.md にある通り、
  1つのハンドラーディレクトリ（`accounts/`）を、責務ごとに複数のディレクトリに分けることを指します。

  例えば、将来 `accounts/` に編集機能やパスワード変更機能が追加されてフィールドが増えた場合：

  ```
  # 分割前
  accounts/
  ├── handler.go  (9+フィールド、肥大化)
  ├── new.go
  ├── create.go
  ├── edit.go
  └── update.go

  # 分割後
  accounts/          (作成に必要な依存のみ)
  ├── handler.go
  ├── new.go
  └── create.go

  account_settings/  (編集に必要な依存のみ)
  ├── handler.go
  ├── edit.go
  └── update.go
  ```

  ただし、現状は8フィールドで上限ぴったりであり、コンストラクタ引数の `userRepo` と `profileRepo` は
  Handler のフィールドではなく validator の初期化にのみ使われているため、設計上は良好です。
  今後フィールドが増えるタイミングで検討すれば十分です。

  **対応**: 現状維持（修正不要）

### `go/internal/ratelimit/limiter.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力（日本語/英語の統一）

**問題点・改善提案**:

- **[@go/CLAUDE.md#コメントのガイドライン]**: `Check` メソッド内のバリデーションエラーメッセージが英語になっている。内部エラーメッセージは日本語で統一するガイドラインがある（`return fmt.Errorf("日本語メッセージ: %w", err)` のパターン）。

  ```go
  // 問題のあるコード
  return nil, fmt.Errorf("key is required")
  return nil, fmt.Errorf("limit must be positive")
  return nil, fmt.Errorf("window must be positive")
  return nil, fmt.Errorf("failed to increment rate limit: %w", err)
  ```

  **修正案**:

  ```go
  return nil, fmt.Errorf("keyは必須です")
  return nil, fmt.Errorf("limitは正の値である必要があります")
  return nil, fmt.Errorf("windowは正の値である必要があります")
  return nil, fmt.Errorf("レート制限カウンターのインクリメントに失敗: %w", err)
  ```

  **対応方針**:
  - [x] 修正案の通り日本語に変更する
  - [ ] 英語のまま（ratelimitパッケージは汎用的なため英語を維持）
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

サインアップ機能の実装は、作業計画書の要件を網羅的に満たしており、ガイドラインへの準拠度も高い。

**良かった点**:

- **アーキテクチャの一貫性**: 3層アーキテクチャの依存関係ルールが正しく守られている。Handler → UseCase → Repository の依存方向が適切で、Query への直接依存もない
- **セキュリティ対策**: CSRF トークン、Turnstile Bot 対策、bcrypt パスワードハッシュ、レート制限がすべて適切に実装されている
- **バリデーション設計**: `validator.go` パターンに従い、形式バリデーション + 状態バリデーションが適切に統合されている。早期リターンでネストが浅く保たれている
- **国際化**: すべてのユーザー向けメッセージが `templates.T()` で国際化されている
- **テストの充実**: 実装コード +2066 行に対しテストコード +2204 行と、十分なテストが書かれている。正常系・異常系の両方が網羅されている
- **WithTx パターン**: `CreateAccountUsecase` でのトランザクション管理が正しく実装されている（`defer func() { _ = tx.Rollback() }()` パターン）
- **エラーハンドリング**: 内部エラーは `slog.ErrorContext` でログ出力し、ユーザーには一般的なエラーメッセージのみ表示する方針が徹底されている
- **作業計画書との整合性**: メール入力 → 確認コード → アカウント作成のフロー、DB テーブル設計、バリデーションルール（atname形式・予約語チェック・パスワード長制限）すべてが計画書通りに実装されている

**指摘事項**:

- accounts ハンドラーの依存性が上限（8個）に達しており、今後の拡張時に注意が必要（推奨対応）
- ratelimit パッケージの内部エラーメッセージが英語になっている（軽微）
