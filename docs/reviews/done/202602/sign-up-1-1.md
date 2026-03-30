# コードレビュー: sign-up-1-1

## レビュー情報

| 項目               | 内容        |
| ------------------ | ----------- |
| レビュー日         | 2026-02-03  |
| 対象ブランチ       | sign-up-1-1 |
| ベースブランチ     | sign-up     |
| 変更ファイル数     | 22 ファイル |
| 変更行数（実装）   | +1000 行    |
| 変更行数（テスト） | +400 行     |

## 参照するガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - プロジェクト全体のガイドライン
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版固有の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/model/profile.go`
- [x] `go/internal/model/user_profile.go`
- [x] `go/internal/repository/actor_repository.go`
- [x] `go/internal/repository/profile_repository.go`
- [x] `go/internal/repository/user_profile_repository.go`
- [x] `go/internal/repository/user_repository.go`
- [x] `go/internal/query/queries/profiles.sql`
- [x] `go/internal/query/queries/user_profiles.sql`
- [x] `go/internal/query/queries/users.sql`
- [x] `go/internal/query/queries/actors.sql`

### テストファイル

- [x] `go/internal/repository/profile_repository_test.go`
- [x] `go/internal/repository/user_profile_repository_test.go`

### 設定・その他

- [x] `docs/designs/1_doing/sign-up.md`
- [x] `docs/designs/2_todo/rate-limiting.md`
- [x] `go/go.mod`
- [x] `go/go.sum`
- [x] `go/internal/query/actors.sql.go` (自動生成)
- [x] `go/internal/query/models.go` (自動生成)
- [x] `go/internal/query/profiles.sql.go` (自動生成)
- [x] `go/internal/query/querier.go` (自動生成)
- [x] `go/internal/query/user_profiles.sql.go` (自動生成)
- [x] `go/internal/query/users.sql.go` (自動生成)

## ファイルごとのレビュー結果

### `go/internal/model/profile.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Model層の設計
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Model と Repository の 1:1 関係

**問題点・改善提案**:

- 問題なし。既存の `model/user.go`, `model/actor.go` と同様の構造で実装されている。

### `go/internal/model/user_profile.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Model層の設計

**問題点・改善提案**:

- 問題なし。シンプルで適切な構造。

### `go/internal/repository/actor_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Repository層の設計
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - コーディング規約

**問題点・改善提案**:

- 問題なし。既存の `session_repository.go` と同様のパターンで実装されている。
- `NewActorRepository`, `WithTx`, `GetByID`, `GetByUserID`, `Create` メソッドが適切に実装されている。
- `ErrNotFound` の処理も正しい。

### `go/internal/repository/profile_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Repository層の設計
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - コーディング規約

**問題点・改善提案**:

- 問題なし。
- `toModel` ヘルパーメソッドを使用して変換ロジックを共通化している点が良い。
- `sql.NullTime` から `*time.Time` への変換が適切に処理されている。

### `go/internal/repository/user_profile_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Repository層の設計

**問題点・改善提案**:

- 問題なし。他のRepositoryと同様のパターンで実装されている。

### `go/internal/repository/user_repository.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Repository層の設計

**問題点・改善提案**:

- 問題なし。新規追加された `ExistsByEmail`, `Create` メソッドが適切に実装されている。

### `go/internal/query/queries/profiles.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Query ファイルの命名

**問題点・改善提案**:

- 問題なし。
- `GetProfileByID`, `GetProfileByAtname`, `ExistsProfileByAtname`, `CreateProfile` が適切に定義されている。
- `LIMIT 1` が適切に使用されている。

### `go/internal/query/queries/user_profiles.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Query ファイルの命名

**問題点・改善提案**:

- 問題なし。

### `go/internal/query/queries/users.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Query ファイルの命名

**問題点・改善提案**:

- 問題なし。新規追加された `ExistsUserByEmail`, `CreateUser` が適切に定義されている。

### `go/internal/query/queries/actors.sql`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - Query ファイルの命名

**問題点・改善提案**:

- 問題なし。

### `go/internal/repository/profile_repository_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - テスト戦略
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - テストのベストプラクティス

**問題点・改善提案**:

- 問題なし。
- `t.Parallel()` を使用した並行テスト
- `testutil.SetupTestDB(t)` を使用したトランザクション分離
- テーブル駆動テストではないが、サブテストを使用して正常系・異常系をテスト
- `testutil.NewProfileBuilder` を使用したテストデータ作成

### `go/internal/repository/user_profile_repository_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - テスト戦略

**問題点・改善提案**:

- 問題なし。`profile_repository_test.go` と同様の良いパターンで実装されている。

### `docs/designs/1_doing/sign-up.md`

**ステータス**: OK

**チェックしたガイドライン**:

- 設計書の内容確認

**問題点・改善提案**:

- 問題なし。タスク1-1が完了としてマークされている。

## 総合評価

**評価**: Approve

**総評**:

このPRはサインアップ機能実装のフェーズ1-1として、Profile、UserProfile、Actor、Userのリポジトリ層を実装しています。

**良かった点**:

1. **既存パターンとの一貫性**: `session_repository.go` や `email_confirmation_repository.go` と同様の設計パターンで実装されており、コードベース全体の一貫性が保たれている
2. **適切なエラーハンドリング**: `sql.ErrNoRows` を `repository.ErrNotFound` に変換する処理が全てのRepositoryで統一されている
3. **WithTxパターンの実装**: トランザクション対応のための `WithTx` メソッドが全てのRepositoryに実装されている
4. **テストカバレッジ**: 正常系・異常系の両方がテストされている
5. **テストヘルパーの活用**: `testutil.NewProfileBuilder`, `testutil.NewUserBuilder` などを活用している
6. **アーキテクチャガイドラインの遵守**: Model と Repository の 1:1 関係、依存関係のルールが守られている

**注意点（修正不要）**:

- ActorRepositoryのテストファイルが今回のPRに含まれていないが、設計書によるとフェーズ1-1のスコープはProfile, UserProfileのリポジトリ実装なので問題なし

---

## 質問と回答

（特に質問はありません）
