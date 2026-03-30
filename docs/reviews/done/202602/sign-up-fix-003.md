# コードレビュー: sign-up-fix

## レビュー情報

| 項目                       | 内容                          |
| -------------------------- | ----------------------------- |
| レビュー日                 | 2026-02-26                    |
| 対象ブランチ               | sign-up-fix                   |
| ベースブランチ             | go                            |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md |
| 変更ファイル数             | 73 ファイル（Go 関連）        |
| 変更行数（実装）           | +5969 / -141 行               |
| 変更行数（テスト）         | +2708 / -32 行                |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 前回レビュー（sign-up-fix-002）からの修正確認

前回のレビューで指摘された 6 件はすべて修正済みであることを確認しました：

1. **FormErrors nil チェック**: `sign_up/create.go` に `result.FormErrors != nil &&` 追加済み
2. **レート制限テスト**: `sign_up/handler_test.go` と `accounts/handler_test.go` に `TestCreate_RateLimitExceeded` 追加済み
3. **validator_test.go パッケージ名**: `sign_up` と `accounts` の両方を内部パッケージに変更済み
4. **`no-reply` 削除**: `reservedAtnames` から `"no-reply"` を削除し、`"noreply"` のみ残した
5. **パスワード最小長**: `utf8.RuneCountInString()` に変更済み（最大長は `len()` を維持）
6. **rate_limit_repository_test.go パッケージ名**: `package repository_test` に変更済み

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/db/migrations/20260204100000_create_rate_limits.sql`
- [x] `go/db/schema.sql`
- [x] `go/internal/auth/password.go`
- [x] `go/internal/handler/accounts/handler.go`
- [x] `go/internal/handler/accounts/new.go`
- [x] `go/internal/handler/accounts/create.go`
- [x] `go/internal/handler/accounts/validator.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/handler/sign_up/new.go`
- [x] `go/internal/handler/sign_up/create.go`
- [x] `go/internal/handler/sign_up/validator.go`
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
- [x] `go/internal/repository/rate_limit_repository.go`
- [x] `go/internal/repository/user_profile_repository.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/templates/pages/accounts/new.templ`
- [x] `go/internal/templates/pages/accounts/new_templ.go`（自動生成）
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`（自動生成）
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
- [x] `go/CLAUDE.md`
- [x] `go/Dockerfile.dev`
- [x] `go/Makefile`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/mise.toml`
- [x] `go/docs/architecture-guide.md`
- [x] `go/docs/handler-guide.md`
- [x] `go/docs/templ-guide.md`
- [x] `go/docs/validation-guide.md`

## ファイルごとのレビュー結果

前回レビュー（sign-up-fix-002）で指摘された問題はすべて修正済みです。今回の再レビューでは、前回未チェックだったファイルも含め全ファイルを確認しました。

問題のあるファイルはありませんでした。

## 設計改善の提案

### `go/db/migrations/20260204100000_create_rate_limits.sql`: 冗長なインデックスの削除

**ステータス**: 要確認

**現状**:

`rate_limits` テーブルに `UNIQUE(key, window_start)` 制約と `CREATE INDEX idx_rate_limits_key_window_start ON rate_limits(key, window_start)` の両方が定義されている。`UNIQUE` 制約は暗黙的に B-tree インデックスを作成するため、明示的なインデックスは冗長であり、書き込みオーバーヘッドが発生する。

```sql
-- 現状: UNIQUE制約とインデックスが重複
UNIQUE(key, window_start)
CREATE INDEX idx_rate_limits_key_window_start ON rate_limits(key, window_start);
CREATE INDEX idx_rate_limits_window_start ON rate_limits(window_start);
```

**提案**:

`idx_rate_limits_key_window_start` インデックスを削除する。`idx_rate_limits_window_start`（`window_start` 単独）は `DeleteOldRateLimits` クエリで使用されるため残す。

```sql
-- 提案: 冗長なインデックスを削除
UNIQUE(key, window_start)
-- idx_rate_limits_key_window_start は削除（UNIQUE制約が暗黙的にカバー）
CREATE INDEX idx_rate_limits_window_start ON rate_limits(window_start);
```

**メリット**:

- 書き込みパフォーマンスの微改善（インデックス更新が 1 つ減る）
- スキーマがよりクリーンになる

**トレードオフ**:

- 既にマイグレーション済みの場合、別マイグレーションが必要
- パフォーマンスへの影響は微小

**対応方針**:

- [x] 別マイグレーションでインデックスを削除する
- [ ] 現状のまま（影響が微小なため）
- [ ] その他（下の回答欄に記入）

**回答**:

```
（ここに回答を記入）
```

## 総合評価

**評価**: Approve

**総評**:

前回レビュー（sign-up-fix-002）で指摘された 6 件の問題がすべて修正されていることを確認しました。修正内容は適切で、ガイドラインに沿っています。

今回の再レビューでは、前回未チェックだったファイルも含め全 73 ファイルを包括的にレビューしました。新たなガイドライン違反や設計との乖離は見つかりませんでした。

**良かった点**:

- 前回の指摘 6 件がすべて正確に修正されている
- 3 層アーキテクチャの依存関係ルールが正確に守られている（Handler/UseCase が Query に直接依存していない）
- セキュリティ対策（CSRF、Turnstile、レート制限、bcrypt、入力バリデーション）が包括的
- I18n が完全対応（ja.toml と en.toml が対称的に定義）
- テンプレートが既存パターン（sign_in）と一致し、構造体ベースの引数パターンを使用
- テストカバレッジが高く、正常系・異常系・レート制限を網羅
- WithTx パターンによるトランザクション管理が正確
- 設計書の全フェーズ（1-1 〜 5-1）の要件が完全に実装されている

**設計改善の提案**: 1 件（冗長なインデックス、影響は微小）
