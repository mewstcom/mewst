# コードレビュー: handler-3-1

## レビュー情報

| 項目                       | 内容                                            |
| -------------------------- | ----------------------------------------------- |
| レビュー日                 | 2026-03-30                                      |
| 対象ブランチ               | handler-3-1                                     |
| ベースブランチ             | handler-2-3                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md  |
| 変更ファイル数             | 14 ファイル                                     |
| 変更行数（実装）           | +15 / -19 行                                    |
| 変更行数（テスト）         | +18 / -13 行                                    |
| 変更行数（ドキュメント）   | +60 / -61 行（Oxfmt フォーマット + タスク更新） |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### 実装ファイル

- [x] `go/cmd/server/main.go`
- [x] `go/internal/handler/sign_in/handler.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/accounts/create.go`
- [x] `go/internal/usecase/create_session.go`

### テストファイル

- [x] `go/internal/handler/sign_in/handler_test.go`
- [x] `go/internal/handler/accounts/handler_test.go`
- [x] `go/internal/usecase/create_session_test.go`

### ドキュメント

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`
- [x] `go/docs/validation-guide.md`
- [x] `docs/reviews/done/202603/handler-1-1-005.md`
- [x] `docs/reviews/done/202603/handler-2-1-001.md`
- [x] `docs/reviews/done/202603/handler-2-2-001.md`
- [x] `docs/reviews/done/202603/handler-2-3-001.md`

## ファイルごとのレビュー結果

問題のあるファイルはありません。すべてのファイルがガイドラインに適合しています。

## 設計との整合性チェック

作業計画書タスク **3-1** の要件をすべて確認しました：

| 要件                                                                 | 状態 |
| -------------------------------------------------------------------- | ---- |
| `CreateSessionUsecase` を拡張し、Actor の取得を UseCase 内に移動する | ✅   |
| `sign_in/handler.go` から `actorRepo` フィールドを削除               | ✅   |
| `sign_in/create.go` を UseCase 経由に修正                            | ✅   |
| `main.go` の更新（CreateSessionUsecase に ActorRepository を注入）   | ✅   |
| テスト更新                                                           | ✅   |
| 作業計画書のタスクチェックボックスを `[x]` に更新                    | ✅   |

### 詳細な確認結果

**1. CreateSessionUsecase の拡張（`go/internal/usecase/create_session.go`）**:

- `actorRepo *repository.ActorRepository` フィールドが追加されている
- `NewCreateSessionUsecase` の引数に `actorRepo` が追加されている
- `CreateSessionInput` の `ActorID` が `UserID` に変更されている
- `Execute` メソッド内で `actorRepo.GetByUserID` を呼び出してアクターを取得している
- エラーは `fmt.Errorf("アクターの取得に失敗: %w", err)` で適切にラップされている

**2. sign_in/handler.go の変更**:

- `actorRepo` フィールドが削除されている
- `repository` パッケージの import が除去されている（Handler → Repository の直接依存が完全に排除）
- `NewHandler` のシグネチャから `actorRepo` 引数が削除されている

**3. sign_in/create.go の変更**:

- `actorRepo.GetByUserID` の呼び出しが削除されている
- `CreateSessionInput` に `UserID: user.ID` を渡すように変更されている
- 不要になったエラーハンドリングブロック（8 行）が削除され、コードがシンプルになっている

**4. accounts/create.go の変更**:

- `CreateSessionInput.ActorID` → `CreateSessionInput.UserID` に変更
- `accountResult.Actor.ID` → `accountResult.Actor.UserID` に変更（UseCase の API 変更に追従）

**5. テストの更新**:

- `create_session_test.go`: 3 つのテスト関数すべてで `actorRepo` を作成し UseCase に注入するように変更。入力が `ActorID` から `UserID` に変更
- `sign_in/handler_test.go`: `createSessionUC` の初期化に `actorRepo` を追加。`NewHandler` 呼び出しから `actorRepo` を削除
- `accounts/handler_test.go`: `createSessionUC` の初期化に `actorRepo` を追加

**6. ドキュメントの変更**:

- `validation-guide.md`: コード例の `ActorID: result.User.ActorID` を `UserID: result.User.ID` に修正（API 変更に追従）
- 過去のレビュードキュメント: Oxfmt によるテーブルフォーマットの自動修正のみ（内容変更なし）

### アーキテクチャ準拠の確認

| ルール                                        | 状態 |
| --------------------------------------------- | ---- |
| Handler → Repository 直接依存の除去           | ✅   |
| Handler → UseCase 経由のデータアクセス        | ✅   |
| UseCase → Repository の依存方向が正しい       | ✅   |
| Handler パッケージから repository import 除去 | ✅   |
| UseCase の単一責任（Execute のみ）            | ✅   |
| エラーメッセージが日本語（開発者向け）        | ✅   |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

タスク 3-1（sign_in ハンドラーの actorRepo 依存を除去）が正確に実装されています。

- `CreateSessionUsecase` への Actor 取得ロジックの移動は、作業計画書の設計通りに実装されている
- `sign_in/handler.go` から `repository` パッケージの import が完全に除去され、Handler → Repository の直接依存が排除されている
- `accounts/create.go` の UseCase API 変更への追従も漏れなく行われている
- テストコードも API 変更に合わせて適切に更新されている
- ドキュメント（validation-guide.md のコード例）も API 変更に合わせて更新されている
- 変更は小さく焦点が絞られており、レビューしやすい
