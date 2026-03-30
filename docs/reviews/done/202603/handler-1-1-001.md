# コードレビュー: handler-1-1

## レビュー情報

| 項目                       | 内容                                           |
| -------------------------- | ---------------------------------------------- |
| レビュー日                 | 2026-03-30                                     |
| 対象ブランチ               | handler-1-1                                    |
| ベースブランチ             | error-3-1                                      |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md |
| 変更ファイル数             | 5 ファイル                                     |
| 変更行数（実装）           | +86 / -139 行（ドキュメントのみ）              |
| 変更行数（テスト）         | +0 / -0 行                                     |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### ドキュメント

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`（新規追加: 作業計画書）
- [x] `go/CLAUDE.md`
- [x] `go/docs/architecture-guide.md`
- [ ] `go/docs/handler-guide.md`
- [ ] `go/docs/validation-guide.md`

## ファイルごとのレビュー結果

### `go/docs/handler-guide.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - 自己参照（ドキュメントの内部整合性）
- [作業計画書](/workspace/docs/plans/1_doing/handler-usecase-refactor.md) - タスク 1-1 の仕様

**問題点・改善提案**:

- **[作業計画書#設計]**: 実装例セクション（旧 L457-L506 付近）から `password_reset/validator.go` のコード例が削除されたが、この例は validation-guide.md に移動されていない。handler-guide.md から削除するだけでなく、validation-guide.md 側のコード例が新しい命名規則に対応していることを確認すべき（後述の validation-guide.md の問題を参照）

  **修正案**:

  validation-guide.md 側のコード例を新しい命名規則に更新すれば、handler-guide.md からの削除は正当。validation-guide.md の修正で対応する。

  **対応方針**:
  - [x] validation-guide.md の修正と合わせて対応する（handler-guide.md 自体は追加修正不要）
  - [ ] handler-guide.md にも新しい命名規則での簡易例を追加する
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/docs/validation-guide.md`

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - 自己参照
- [作業計画書](/workspace/docs/plans/1_doing/handler-usecase-refactor.md) - タスク 1-1 の仕様
- [@go/CLAUDE.md#バリデーション](/workspace/go/CLAUDE.md) - バリデーション命名規則

**問題点・改善提案**:

- **[作業計画書#タスク1-1 / CLAUDE.md#バリデーション]**: 概要セクション（L1-L36）と Handler の依存性セクション（L345+）は新しい命名規則（`{Handler}{Action}Validator`、`internal/validator/` パッケージ）に更新されているが、ファイルの大部分（実装例、テスト例、ベストプラクティス）が旧パターンのまま残っている。

  以下の箇所で旧パターンが使用されている：

  **1. 「状態バリデーションの配置場所」セクション（L40, L46, L49, L68）**:

  ```
  # L40
  状態バリデーションは `validator.go` または UseCase のどちらかに配置します。
  # L46
  | 不要 | validator.go | UseCase をシンプルに保つため |
  # L49
  **validator.go で行うべき検証**:
  # L68
  検証自体は validator.go で行い、成功後に UseCase を呼び出す。
  ```

  **2. コード例のファイルパスとパッケージ名（L97-L189, L195-L248, L253-L300）**:

  ```go
  // L100: 旧パス
  // internal/handler/sign_in/validator.go
  package sign_in

  // L195: 旧パス
  // internal/handler/password_reset/validator.go
  package password_reset

  // L253: 旧パス
  // internal/handler/password/validator.go
  package password
  ```

  **3. コード例の構造体名（L120-L135, L208-L232, L263-L300）**:

  ```go
  // L120: 旧命名
  type CreateValidator struct { ... }
  // → 新命名: SignInCreateValidator

  // L208: 旧命名
  type CreateValidator struct{}
  // → 新命名: PasswordResetCreateValidator

  // L263: 旧命名
  type UpdateValidator struct{}
  // → 新命名: PasswordUpdateValidator
  ```

  **4. テスト例のファイルパス（L400）**:

  ```go
  // internal/handler/sign_in/validator_test.go
  // → internal/validator/sign_in_test.go
  ```

  **5. ベストプラクティスセクション（L543-L559）**:

  ```go
  // ✅ Good: 1つのファイル（validator.go）に統合
  // internal/handler/sign_in/validator.go
  type CreateValidator struct { ... }
  ```

  **修正案**:

  すべてのコード例・テキストを新しい命名規則に統一する。具体的には：
  - ファイルパス: `internal/handler/{name}/validator.go` → `internal/validator/{name}.go`
  - パッケージ名: `package {name}` → `package validator`
  - 構造体名: `{Action}Validator` → `{Handler}{Action}Validator`
  - コンストラクタ: `New{Action}Validator()` → `New{Handler}{Action}Validator()`
  - テキスト中の `validator.go` → `Validator` またはバリデーターパッケージへの適切な表現

  **対応方針**:
  - [x] すべてのコード例・テキストを新しい命名規則に一括更新する
  - [ ] コード例の更新は次のフェーズ（Validator 移動時）に合わせて行う
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

## 設計との整合性チェック

### 作業計画書タスク 1-1 との整合性

作業計画書のタスク 1-1 で要求されている変更内容：

| 要件                                                                     | 状態       | 備考                                                                |
| ------------------------------------------------------------------------ | ---------- | ------------------------------------------------------------------- |
| `architecture-guide.md`: Handler → Repository の依存を禁止するルール追加 | OK         | L183, L228 で正しく更新                                             |
| `architecture-guide.md`: UseCase の役割を書き込み + 読み取りに拡張       | OK         | L401-L462 で UseCase の種類テーブルと読み取り UseCase の例を追加    |
| `architecture-guide.md`: Validator パッケージの分離について記述追加      | OK         | L136 で `internal/validator` を Presentation 層に追加               |
| `architecture-guide.md`: 読み取り UseCase の設計パターンとコード例       | OK         | L430-L462 で `GetActiveEmailConfirmationUsecase` の例を追加         |
| `CLAUDE.md`: 重要な設計原則セクションの更新                              | OK         | L106, L110-L116 で更新                                              |
| `validation-guide.md`: Validator の配置先を `internal/validator/` に変更 | **部分的** | 概要と Handler 依存性セクションのみ更新、コード例は旧パターンのまま |
| `handler-guide.md`: validator.go を標準ファイルから削除                  | OK         | 9 → 8 種類に変更、バリデーター配置例も更新済み                      |

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Request Changes

**総評**:

本 PR はタスク 1-1（アーキテクチャドキュメントの更新）として、Handler → Repository 依存関係の廃止に向けたガイドライン更新を行っている。`go/CLAUDE.md`、`go/docs/architecture-guide.md`、`go/docs/handler-guide.md` の変更は正確かつ一貫性があり、新しいアーキテクチャルールを適切に反映している。

ただし、`go/docs/validation-guide.md` については概要セクションのみ更新され、ファイルの大部分（実装例、テスト例、ベストプラクティス）が旧パターン（`internal/handler/*/validator.go`、`{Action}Validator` 命名）のまま残っている。ドキュメントの前半と後半で異なるパターンが混在しており、読者に混乱を与える状態になっている。この不整合の修正が必要。
