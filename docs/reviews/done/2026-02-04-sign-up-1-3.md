# コードレビュー: sign-up タスク 1-3（レート制限機能）

## レビュー情報

| 項目              | 内容                              |
| ----------------- | --------------------------------- |
| レビュー日        | 2026-02-04                        |
| 対象ブランチ      | sign-up                           |
| ベースブランチ    | main                              |
| 変更ファイル数    | 9 ファイル                        |
| 変更行数（実装）  | +422 / -0 行                      |
| 変更行数（テスト）| +170 / -0 行                      |

## 参照するガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - 全体的なコーディング規約
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版コーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `db/migrations/20260204100000_create_rate_limits.sql`
- [x] `db/schema.sql`
- [x] `internal/query/queries/rate_limits.sql`
- [x] `internal/query/models.go` (自動生成)
- [x] `internal/query/rate_limits.sql.go` (自動生成)
- [x] `internal/ratelimit/limiter.go`

### テストファイル

- [x] `internal/ratelimit/limiter_test.go`

### ドキュメント

- [x] `docs/designs/1_doing/sign-up.md` (チェックボックス更新)

## ファイルごとのレビュー結果

### `db/migrations/20260204100000_create_rate_limits.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#カラム定義のガイドライン](/workspace/go/CLAUDE.md) - VARCHAR、TIMESTAMP WITH TIME ZONE

**問題点・改善提案**:

- 問題なし
- VARCHAR（長さ指定なし）を使用している ✅
- TIMESTAMP WITH TIME ZONE を使用している ✅
- ULID（uuid型）を主キーに使用している ✅
- 適切なインデックスが作成されている ✅

### `internal/query/queries/rate_limits.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#Query ファイルの命名](/workspace/go/docs/architecture-guide.md) - テーブル名ベース

**問題点・改善提案**:

- 問題なし
- 単一テーブルに対するCRUD操作なので、テーブル名ベースの命名（`rate_limits.sql`）は適切 ✅
- SQLコメントが日本語で記述されている ✅
- UPSERTパターンが適切に実装されている ✅

### `internal/ratelimit/limiter.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#重要なルール](/workspace/go/docs/architecture-guide.md) - Query への依存は Repository のみ
- [@go/CLAUDE.md#コメントのガイドライン](/workspace/go/CLAUDE.md) - コメント規約

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#重要なルール]**: `ratelimit.Limiter` が `internal/query` に直接依存している

  ```go
  // 現在の実装
  type Limiter struct {
      q *query.Queries
  }
  ```

  アーキテクチャガイドでは「Query への依存は Repository のみ」とされていますが、`ratelimit` パッケージは `internal/query` に直接依存しています。

  **考察**:
  - `ratelimit` は「インフラストラクチャユーティリティ」として Domain/Infrastructure 層に位置づけることもできる
  - 設計書では「Wikino の実装を参考」とあり、Wikino でも同じパターンを使用している
  - 単一テーブルへの単純な操作であり、Model への変換が不要なため Repository を経由するメリットが少ない

  → **質問として挙げる**

- それ以外は問題なし:
  - パッケージコメントが適切に記述されている ✅
  - エラーメッセージが日本語で記述されている ✅
  - `WithTx` メソッドでトランザクション対応している ✅

### `internal/ratelimit/limiter_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - 実データベーステスト、テーブル駆動テスト

**問題点・改善提案**:

- 問題なし
- 実データベースを使用したテスト ✅
- `testutil.SetupTestDB(t)` を使用している ✅
- `t.Parallel()` で並行テストを有効化している ✅
- 正常系・異常系を網羅的にテストしている ✅
- テスト名が日本語で分かりやすい ✅

### `internal/query/models.go` / `internal/query/rate_limits.sql.go`

**ステータス**: OK（自動生成）

**問題点・改善提案**:

- sqlc により自動生成されたファイルのため、レビュー対象外

### `db/schema.sql`

**ステータス**: OK

**問題点・改善提案**:

- マイグレーション実行により自動更新されたファイル
- rate_limits テーブルが正しく追加されている ✅

## 総合評価

**評価**: Comment

**総評**:

レート制限機能の実装は全体的に良好です。以下の点が評価できます：

**良い点**:
- カラム定義ガイドラインに従っている（VARCHAR長さ指定なし、TIMESTAMP WITH TIME ZONE）
- Wikino の実装を参考にした一貫性のある設計
- 包括的なテストカバレッジ（正常系・異常系を網羅）
- 並行テストによる高速なテスト実行
- 適切な日本語コメント

**確認が必要な点**:
- `ratelimit` パッケージが `internal/query` に直接依存している点について、アーキテクチャガイドとの整合性を確認したい

実装自体は問題なく動作し、テストも通過しているため、軽微な指摘のみです。

---

## 質問と回答

### Q1: ratelimit パッケージの Query 直接依存について

**種別**: 推奨

**背景**:

アーキテクチャガイド（[@go/docs/architecture-guide.md#重要なルール](/workspace/go/docs/architecture-guide.md)）では「Query への依存は Repository のみ」と記載されています。

しかし、`ratelimit.Limiter` は `internal/query.Queries` に直接依存しています：

```go
type Limiter struct {
    q *query.Queries
}
```

以下の選択肢が考えられます：

**選択肢**:

- [ ] 選択肢A: 現状維持 - `ratelimit` はインフラストラクチャユーティリティとして例外扱いし、Query への直接依存を許容する（Wikino と同じパターン）
- [x] 選択肢B: Repository 層を追加 - `repository.RateLimitRepository` を作成し、アーキテクチャガイドに完全準拠する
- [ ] その他（下の回答欄に記入）

**回答**:

```
Mewstの修正をしつつ、Wikino のほうで同様の修正ができるようにするために
@/wikino/docs/designs/template.md をコピーし、テンプレートに沿って設計書を作成してください。
```

**対応完了**:

- **Mewst**: `repository.RateLimitRepository` を作成し、`ratelimit.Limiter` が Repository 経由でデータアクセスするように修正済み
  - `internal/repository/rate_limit_repository.go` (新規)
  - `internal/repository/rate_limit_repository_test.go` (新規)
  - `internal/ratelimit/limiter.go` (修正)
  - `internal/ratelimit/limiter_test.go` (修正)
- **Wikino**: 設計書を作成済み
  - `/wikino/docs/designs/2_todo/go-rate-limit-repository.md`
