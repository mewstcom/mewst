# コードレビュー: handler-3-3

## レビュー情報

| 項目                       | 内容                                                     |
| -------------------------- | -------------------------------------------------------- |
| レビュー日                 | 2026-03-30                                               |
| 対象ブランチ               | handler-3-3                                              |
| ベースブランチ             | handler-3-2                                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md            |
| 変更ファイル数             | 5 ファイル                                               |
| 変更行数（実装）           | +24 / -22 行                                             |
| 変更行数（テスト）         | +3 / -2 行                                               |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/accounts/handler.go`
- [ ] `go/internal/handler/accounts/new.go`

### テストファイル

- [x] `go/internal/handler/accounts/handler_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

### `go/internal/handler/accounts/new.go`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md#レイヤー間の依存関係](/workspace/go/docs/architecture-guide.md) - Handler は Repository に直接依存しない
- [作業計画書#depguardによる強制](/workspace/docs/plans/1_doing/handler-usecase-refactor.md) - Handler → Repository の依存を禁止

**問題点・改善提案**:

- **[@go/docs/architecture-guide.md#重要なルール]**: `new.go` が `repository` パッケージを引き続き import している（`repository.ErrNotFound` のエラー比較に使用）

  ```go
  // 現在のコード (new.go:13)
  "github.com/mewstcom/mewst/go/internal/repository"
  ```

  ```go
  // 現在のコード (new.go:78)
  if errors.Is(err, repository.ErrNotFound) {
  ```

  タスク 3-3 の目的は `emailConfirmationRepo` フィールドの除去であり、これは達成されている。ただし `repository.ErrNotFound` の import が残っているため、フェーズ 4（depguard で Handler → Repository の import を禁止）を実施する前にこの依存を解消する必要がある。

  なお、同じパターンが `password/edit.go`、`email_confirmation/new.go` にも存在する（タスク 3-2 で導入）。

  **修正案**:

  UseCase が `repository.ErrNotFound` をラップせず、Handler が認識可能な別のエラー（例: `usecase.ErrNotFound`）を返すか、見つからない場合は `nil` output を返すように変更する。

  ```go
  // 案A: UseCase が独自のエラーを返す
  // usecase パッケージに ErrNotFound を定義
  var ErrNotFound = errors.New("not found")

  // GetSucceededEmailConfirmationUsecase.Execute 内で変換
  if errors.Is(err, repository.ErrNotFound) {
      return nil, ErrNotFound
  }

  // Handler 側
  if errors.Is(err, usecase.ErrNotFound) {
      return nil, nil
  }
  ```

  ```go
  // 案B: UseCase が nil output を返す（not found は正常系として扱う）
  // GetSucceededEmailConfirmationUsecase.Execute 内
  if errors.Is(err, repository.ErrNotFound) {
      return nil, nil  // not found は nil を返す
  }

  // Handler 側
  output, err := h.getSucceededEmailConfirmationUC.Execute(...)
  if err != nil {
      return nil, err
  }
  if output == nil {
      return nil, nil
  }
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] フェーズ 4 の前に対応する（対象: `new.go`, `password/edit.go`, `email_confirmation/new.go` を一括修正）
  - [ ] フェーズ 4 のタスク 4-1 に含める
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

タスク 3-3 の目的である「accounts ハンドラーの `emailConfirmationRepo` 依存を除去」は正しく達成されている。

- `handler.go` から `repository` パッケージの import と `emailConfirmationRepo` フィールドが完全に除去されている
- `new.go` で `emailConfirmationRepo.GetSucceededByID()` の呼び出しが `getSucceededEmailConfirmationUC.Execute()` に置き換えられている
- エラーチェックが `err == repository.ErrNotFound` から `errors.Is(err, repository.ErrNotFound)` に改善されている（ラップされたエラーにも対応）
- `main.go` と `handler_test.go` の更新も適切
- `create.go` は `repository` を import しておらず、問題なし
- 作業計画書のタスク 3-3 のチェックボックスも更新されている

唯一の指摘は `new.go` に残る `repository.ErrNotFound` の import で、フェーズ 4 の depguard 強制前に対応方針を決める必要がある。ただしこれはタスク 3-3 のスコープ外であり、タスク 3-2 で導入された共通パターンであるため、ブロッカーではない。
