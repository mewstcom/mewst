# コードレビュー: handler-3-2

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-30                                     |
| 対象ブランチ               | handler-3-2                                    |
| ベースブランチ             | handler-3-1                                    |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 13 ファイル                                    |
| 変更行数（実装）           | +128 / -28 行                                  |
| 変更行数（テスト）         | +214 / -4 行                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/usecase/get_active_email_confirmation.go`
- [x] `go/internal/usecase/get_succeeded_email_confirmation.go`
- [x] `go/internal/handler/email_confirmation/handler.go`
- [x] `go/internal/handler/email_confirmation/new.go`
- [x] `go/internal/handler/password/handler.go`
- [x] `go/internal/handler/password/edit.go`
- [x] `go/internal/handler/password/update.go`
- [x] `go/cmd/server/main.go`

### テストファイル

- [x] `go/internal/usecase/get_active_email_confirmation_test.go`
- [x] `go/internal/usecase/get_succeeded_email_confirmation_test.go`
- [x] `go/internal/handler/email_confirmation/handler_test.go`
- [x] `go/internal/handler/password/handler_test.go`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。全ファイルがガイドラインに準拠しています。

## 設計との整合性チェック

### 作業計画書タスク 3-2 の要件確認

| 要件                                                                          | 状態 |
| ----------------------------------------------------------------------------- | ---- |
| `GetActiveEmailConfirmationUsecase` を作成（`email_confirmation/new.go` 用）  | ✅   |
| `GetSucceededEmailConfirmationUsecase` を作成（`password/edit.go` 用）        | ✅   |
| `email_confirmation/handler.go` から `emailConfirmationRepo` フィールドを削除 | ✅   |
| `password/handler.go` から `emailConfirmationRepo` フィールドを削除           | ✅   |
| `main.go` の更新                                                              | ✅   |
| テスト追加・更新                                                              | ✅   |

すべての要件が実装されています。設計との乖離はありません。

### 補足事項

`email_confirmation/new.go`、`password/edit.go`、`password/update.go` は引き続き `repository` パッケージを `repository.ErrNotFound` のためにインポートしています。これはタスク 4-1（depguard による Handler → Repository 禁止）で対処される既知の課題であり、本 PR のスコープ外です。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

作業計画書タスク 3-2 の要件をすべて満たしたクリーンなリファクタリングです。

- **UseCase の実装**: `GetActiveEmailConfirmationUsecase` と `GetSucceededEmailConfirmationUsecase` は、既存のアーキテクチャガイドに記載された読み取り UseCase のパターンに忠実に従っている
- **命名規則**: ファイル名 `{action}_{entity}.go`、構造体名 `{Action}{Entity}Usecase` の規則に準拠
- **テストの網羅性**: 正常系（Success）、異常系（NotFound）、境界値（Expired / NotSucceeded）の 3 パターンをカバー
- **Handler の依存除去**: `handler.go` から `repository` パッケージの import を完全に除去し、UseCase 経由に統一
- **既存テストとの一貫性**: `testutil.SetupTestDB(t)` の使用など、既存の usecase テストパターンと一致
