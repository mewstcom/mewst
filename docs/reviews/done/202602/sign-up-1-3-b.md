# コードレビュー: sign-up-1-3（Repository層追加・UseCaseリファクタリング・Worker実装）

## レビュー情報

| 項目               | 内容          |
| ------------------ | ------------- |
| レビュー日         | 2026-02-04    |
| 対象ブランチ       | sign-up-1-3   |
| ベースブランチ     | sign-up       |
| 変更ファイル数     | 13 ファイル   |
| 変更行数（実装）   | +389 / -92 行 |
| 変更行数（テスト） | +620 / -30 行 |

## 参照するガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - 全体的なコーディング規約
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版コーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/repository/rate_limit_repository.go`
- [x] `go/internal/ratelimit/limiter.go`
- [x] `go/internal/usecase/create_email_confirmation.go`
- [x] `go/internal/worker/client.go`
- [x] `go/internal/worker/send_email_confirmation.go`
- [x] `go/cmd/server/main.go`
- [x] `go/.golangci.yml`
- [x] `go/db/migrations/20260204100000_create_rate_limits.sql`

### テストファイル

- [x] `go/internal/repository/rate_limit_repository_test.go`
- [x] `go/internal/ratelimit/limiter_test.go`
- [x] `go/internal/usecase/create_email_confirmation_test.go`
- [x] `go/internal/worker/send_email_confirmation_test.go`
- [x] `go/internal/handler/password_reset/handler_test.go`

### 設定・その他

- [x] `go/internal/query/models.go` (自動生成)
- [x] `go/internal/query/querier.go` (自動生成)
- [x] `go/internal/query/rate_limits.sql.go` (自動生成)
- [x] `go/db/schema.sql` (自動更新)

## ファイルごとのレビュー結果

### `go/internal/repository/rate_limit_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#Repository層のルール](/workspace/go/docs/architecture-guide.md) - Repository設計

**問題点・改善提案**:

- 問題なし
- 前回レビューで指摘した「Queryへの直接依存」が解消されている ✅
- `WithTx`メソッドでトランザクション対応している ✅
- 独自の入出力型（`IncrementParams`, `IncrementResult`）を定義している ✅
- 日本語コメントが適切に記述されている ✅

### `go/internal/repository/rate_limit_repository_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし
- 実データベースを使用したテスト ✅
- `testutil.SetupTestDB(t)` を使用している ✅
- `t.Parallel()` で並行テストを有効化している ✅
- 正常系を網羅的にテストしている ✅
- テスト名が日本語で分かりやすい ✅

### `go/internal/ratelimit/limiter.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#重要なルール](/workspace/go/docs/architecture-guide.md) - 依存関係ルール
- [@go/CLAUDE.md#コメントのガイドライン](/workspace/go/CLAUDE.md) - コメント規約

**問題点・改善提案**:

- 問題なし
- 前回レビューで指摘した「Queryへの直接依存」が解消され、Repository経由になっている ✅
- パッケージコメントが適切に記述されている ✅
- エラーメッセージが英語（Goの慣習に従っている） ✅
- `WithTx` メソッドでトランザクション対応している ✅
- `ErrRateLimitExceeded` センチネルエラーが定義されている ✅
- ヘルパー関数 `IPKey`, `EmailKey` が提供されている ✅

### `go/internal/ratelimit/limiter_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし
- 実データベースを使用したテスト ✅
- `t.Parallel()` で並行テストを有効化している ✅
- 正常系・異常系を網羅的にテストしている ✅
- バリデーションエラーケースをテストしている ✅
- `Allow` メソッドもテストしている ✅
- ヘルパー関数のテストも含まれている ✅

### `go/internal/usecase/create_email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ユースケース](/workspace/go/CLAUDE.md) - UseCase設計
- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力規約

**問題点・改善提案**:

- 問題なし
- `Execute` メソッドのみを公開している（単一責任） ✅
- Repository経由でデータアクセスしている ✅
- `slog.ErrorContext` / `slog.InfoContext` を使用している ✅
- メールテンプレートのレンダリングをWorkerに移動している（設計上の改善） ✅
- ジョブエンキュー失敗時もコードは有効なため、エラーログを出力して続行する適切なエラーハンドリング ✅

### `go/internal/usecase/create_email_confirmation_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし
- 実データベースを使用したテスト ✅
- モック `mockInserter` でジョブエンキューをテスト ✅
- 複数のテストケースを網羅している（日本語/英語ロケール、コード一意性、レコード永続化、イベントタイプ） ✅

### `go/internal/worker/client.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力規約

**問題点・改善提案**:

- 問題なし
- `NewSendEmailConfirmationWorker` を登録している ✅
- `slog.Info` を使用している（コンテキストなしの適切なケース） ✅

### `go/internal/worker/send_email_confirmation.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力規約
- [@go/CLAUDE.md#コメントのガイドライン](/workspace/go/CLAUDE.md) - コメント規約

**問題点・改善提案**:

- 問題なし
- River Job の標準的なインターフェース（`Kind()`, `InsertOpts()`）を実装している ✅
- メールテンプレートのレンダリングをWorker内で行っている（UseCaseからの移動） ✅
- ロケールに基づいて適切なテンプレートを選択している ✅
- `slog.InfoContext` / `slog.ErrorContext` を使用している ✅
- 適切なエラーハンドリング ✅
- `MaxAttempts: 5` でリトライ設定されている ✅

### `go/internal/worker/send_email_confirmation_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし
- テーブル駆動テストを使用している ✅
- 正常系（日本語・英語・未知のロケール）をテストしている ✅
- 異常系（空のメールアドレス）をテストしている ✅
- `NoopSender` を使用してメール送信をモックしている ✅
- `Kind()` と `InsertOpts()` のテストも含まれている ✅

### `go/cmd/server/main.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - ログ出力規約

**問題点・改善提案**:

- 問題なし
- 変更は `emailConfirmationRepo` 変数の削除のみ（軽微）
- 依存関係の整理が適切に行われている ✅

### `go/.golangci.yml`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#重要なルール](/workspace/go/docs/architecture-guide.md) - 依存関係ルール

**問題点・改善提案**:

- 問題なし
- `ratelimit-layer` ルールが追加されている ✅
- `ratelimit` パッケージが Query に直接依存できないルールが設定されている ✅
- 他の層（UseCase, Handler, Middleware, ViewModel, Templates）への依存も禁止されている ✅

### `go/db/migrations/20260204100000_create_rate_limits.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#カラム定義のガイドライン](/workspace/go/CLAUDE.md) - カラム定義

**問題点・改善提案**:

- 問題なし（前回レビューで確認済み）
- VARCHAR（長さ指定なし）を使用している ✅
- TIMESTAMP WITH TIME ZONE を使用している ✅
- ULID（uuid型）を主キーに使用している ✅
- 適切なインデックスが作成されている ✅

### `go/internal/handler/password_reset/handler_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし
- `mockInserter` を使用してジョブエンキューをモックしている ✅
- 既存のテストケースが維持されている ✅

## 総合評価

**評価**: Approve

**総評**:

前回のレビューで指摘した「`ratelimit` パッケージが Query に直接依存している」問題が適切に修正されています。

**良い点**:

1. **アーキテクチャの改善**
   - `RateLimitRepository` が追加され、アーキテクチャガイドに完全準拠
   - `ratelimit` パッケージは Repository 経由でデータアクセスするようになった
   - `golangci-lint` の `depguard` ルールで依存関係を強制

2. **責任の分離**
   - メールテンプレートのレンダリングが UseCase から Worker に移動
   - UseCase は確認コードの生成とレコード作成に集中
   - Worker はメールの生成と送信に集中

3. **包括的なテストカバレッジ**
   - Repository、Limiter、UseCase、Worker すべてにテストが追加
   - 正常系・異常系を網羅的にテスト
   - テーブル駆動テストを適切に使用

4. **ログ出力規約の遵守**
   - すべての箇所で `slog` パッケージを使用
   - コンテキストありの場合は `slog.InfoContext` / `slog.ErrorContext` を使用

**改善が必要な点**:

- なし

すべての変更がガイドラインに従っており、マージ可能です。

---

## 質問と回答

（質問なし）
