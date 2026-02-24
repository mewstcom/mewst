# コードレビュー: sign-up-fix

## レビュー情報

| 項目                         | 内容                                              |
| ---------------------------- | ------------------------------------------------- |
| レビュー日                   | 2026-02-24                                        |
| 対象ブランチ                 | sign-up-fix                                       |
| ベースブランチ               | develop                                           |
| 作業計画書（指定があれば）   | docs/plans/1_doing/sign-up.md                     |
| 変更ファイル数               | 32 ファイル                                       |
| 変更行数（実装）             | +2386 行                                          |
| 変更行数（テスト）           | +1091 行                                          |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/handler/sign_up/new.go`
- [x] `go/internal/handler/sign_up/create.go`
- [x] `go/internal/handler/sign_up/validator.go`
- [x] `go/internal/handler/email_confirmation/handler.go`
- [x] `go/internal/handler/email_confirmation/new.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/validator.go`
- [x] `go/internal/usecase/create_account.go`
- [x] `go/internal/repository/profile_repository.go`
- [x] `go/internal/repository/user_profile_repository.go`
- [x] `go/internal/repository/actor_repository.go`
- [x] `go/internal/ratelimit/limiter.go`
- [x] `go/internal/model/profile.go`
- [x] `go/internal/model/user_profile.go`
- [x] `go/internal/model/actor.go`
- [x] `go/internal/testutil/builder.go`
- [x] `go/internal/testutil/db.go`

### テンプレート・生成ファイル

- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`（自動生成）

### SQL クエリファイル

- [x] `go/internal/query/queries/profiles.sql`
- [x] `go/internal/query/queries/user_profiles.sql`
- [x] `go/internal/query/queries/actors.sql`
- [x] `go/internal/query/queries/rate_limits.sql`

### 国際化ファイル

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`

### テストファイル

- [x] `go/internal/handler/sign_up/handler_test.go`
- [x] `go/internal/handler/sign_up/validator_test.go`
- [x] `go/internal/handler/email_confirmation/handler_test.go`
- [x] `go/internal/handler/email_confirmation/validator_test.go`
- [x] `go/internal/usecase/create_account_test.go`
- [x] `go/internal/ratelimit/limiter_test.go`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計改善の提案

設計改善の提案はありません。

## 設計との整合性チェック

作業計画書（`docs/plans/1_doing/sign-up.md`）との整合性を確認しました。

### 完了済みフェーズ（実装確認済み）

**フェーズ 1: 基盤整備** - 全タスク完了

- **1-1**: リポジトリ層（Profile, UserProfile）が設計通り実装されている。`ActorRepository` も追加されており、設計のディレクトリ構造には明示されていなかったが `create_account.go` で必要なため適切
- **1-2**: `CreateAccountUsecase` が設計通り実装されている。トランザクション管理、Profile → User → UserProfile → Actor の順序での作成、bcrypt ハッシュ化が正しく実装されている。結果型名が設計では `CreateAccountOutput` だが実装では `CreateAccountResult` となっており、これは CLAUDE.md の命名規則（`{Action}{Entity}Result`）に正しく従った結果
- **1-3**: レート制限が設計通り実装されている。Repository パターンを使用した PostgreSQL ベースの実装

**フェーズ 2: サインアップフォーム** - 全タスク完了

- **2-1**: サインアップハンドラーが設計通り実装されている。設計にはなかった `rateLimiter` と `turnstile.Verifier`（インターフェース）が追加されており、セキュリティの向上に寄与
- **2-2**: テンプレートと国際化が設計通り実装されている

**フェーズ 3: メール確認フローの拡張** - 全タスク完了

- **3-1**: `getRedirectPath` で `sign_up` イベントを `/accounts/new` にリダイレクトするよう正しく実装されている

### 未実装フェーズ（想定通り）

- **フェーズ 4**: アカウント作成フォーム（`accounts/` ハンドラー）- 未着手（タスクリストで `[ ]`）
- **フェーズ 5**: ルーティング設定・リバースプロキシ更新 - 未着手（タスクリストで `[ ]`）

### 設計との乖離

設計との乖離はありません。実装は設計に忠実で、追加された要素（レート制限、Turnstile インターフェース化）はいずれも改善です。

## 総合評価

**評価**: Approve

**総評**:

サインアップ機能のフェーズ 1-3 が設計書に沿って正しく実装されています。

**良かった点**:

- **ガイドライン準拠**: ハンドラーの標準ファイル名（9 種類）、バリデーターのパターン（`{Action}Validator`）、ユースケースの命名規則（`{Action}{Entity}Usecase` + 単一 `Execute` メソッド）、Repository の 1:1 パターンがすべて正しく適用されている
- **セキュリティ**: CSRF トークン、Cloudflare Turnstile、レート制限、bcrypt ハッシュ化の 4 重のセキュリティ対策が適切に実装されている
- **テストカバレッジ**: 正常系・異常系（空メール、不正形式、Turnstile 失敗、メールアドレス重複）が網羅されている
- **アーキテクチャ**: 3 層アーキテクチャの依存関係ルールを遵守。Handler → UseCase → Repository の依存方向が正しい。Query への直接依存がない
- **国際化**: すべてのユーザー向けメッセージが `templates.T(ctx, ...)` で国際化されている
- **既存コードとの一貫性**: テストの `SetupTestDB` パターン、ハードコードされたロケール `"ja"` の設定方法など、既存ハンドラー（sign_in, password_reset 等）と完全に統一されている
- **Turnstile のインターフェース化**: 設計書では `*turnstile.Client` だったが、実装では `turnstile.Verifier` インターフェースを使用しており、テスタビリティが向上している
