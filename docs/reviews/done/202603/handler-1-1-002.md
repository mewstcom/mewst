# コードレビュー: handler-1-1

## レビュー情報

| 項目                       | 内容                                                   |
| -------------------------- | ------------------------------------------------------ |
| レビュー日                 | 2026-03-30                                             |
| 対象ブランチ               | handler-1-1                                            |
| ベースブランチ             | error-3-1                                              |
| 作業計画書（指定があれば） | docs/plans/1_doing/handler-usecase-refactor.md         |
| 変更ファイル数             | 4 ファイル（ドキュメントのみ、レビュー・計画書を除く） |
| 変更行数（実装）           | +173 / -226 行                                         |
| 変更行数（テスト）         | +0 / -0 行                                             |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションガイド

## 変更ファイル一覧

### ドキュメント

- [x] `go/CLAUDE.md`
- [ ] `go/docs/architecture-guide.md`
- [ ] `go/docs/handler-guide.md`
- [x] `go/docs/validation-guide.md`

### 設定・その他

- [x] `docs/plans/1_doing/handler-usecase-refactor.md`（作業計画書）
- [x] `docs/reviews/handler-1-1-001.md`（前回レビュー）

## 作業計画書との整合性チェック

タスク 1-1 の要件と変更内容の対応:

| 要件                                                               | 対応状況 |
| ------------------------------------------------------------------ | -------- |
| architecture-guide.md: Handler → Repository 依存禁止ルール追加     | ✅       |
| architecture-guide.md: UseCase の役割を書き込み＋読み取りに拡張    | ✅       |
| architecture-guide.md: Validator パッケージ分離の記述追加          | ✅       |
| architecture-guide.md: 読み取り UseCase の設計パターン・コード例   | ✅       |
| CLAUDE.md: 重要な設計原則セクション更新                            | ✅       |
| validation-guide.md: Validator 配置先を internal/validator/ に変更 | ✅       |
| handler-guide.md: ハンドラーディレクトリから validator.go 削除     | ✅       |

## ファイルごとのレビュー結果

### `go/docs/architecture-guide.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@CLAUDE.md#コメントのガイドライン](/workspace/CLAUDE.md) - コメントのガイドライン

**問題点・改善提案**:

- **[ドキュメント内の一貫性]**: 読み取り UseCase のコード例（440行目付近）で `GetActiveEmailConfirmationUsecase` にコンストラクタ（`NewGetActiveEmailConfirmationUsecase`）が定義されていない。既存の書き込み UseCase の例（`DeleteStripeSubscriberUsecase`）にもコンストラクタはないが、CLAUDE.md のユースケース命名規則では「コンストラクタ: `New{Action}{Entity}Usecase`」と明記されている。新しく追加する読み取り UseCase の例にはコンストラクタも含めたほうが、パターンが明確になる。

  **修正案**:

  読み取り UseCase のコード例にコンストラクタを追加する:

  ```go
  func NewGetActiveEmailConfirmationUsecase(
      emailConfirmationRepo *repository.EmailConfirmationRepository,
  ) *GetActiveEmailConfirmationUsecase {
      return &GetActiveEmailConfirmationUsecase{
          emailConfirmationRepo: emailConfirmationRepo,
      }
  }
  ```

  **対応方針**:
  - [x] コンストラクタを追加する
  - [ ] 現状のまま（既存の書き込み UseCase 例にもないため統一）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/docs/handler-guide.md`

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

**問題点・改善提案**:

- **[ドキュメント内の一貫性 - ルーティング例]**: 「ルーティング登録」セクション（317行目付近）のコード例で `NewHandler` のシグネチャが旧アーキテクチャのままになっている。特に `signInHandler := sign_in.NewHandler(cfg, queries, sessionMgr)` は validator を受け取っておらず、同じドキュメント内の「バリデーターの配置」セクションや validation-guide.md の「Handler の依存性」セクションの例と矛盾している。

  ```go
  // 現在のルーティング例（317行目付近）
  signInHandler := sign_in.NewHandler(cfg, queries, sessionMgr)
  ```

  **修正案**:

  新アーキテクチャに合わせてルーティング例を更新する。少なくとも Validator を注入する形にする:

  ```go
  // ハンドラーの初期化
  signInValidator := validator.NewSignInCreateValidator(userRepo)
  signInHandler := sign_in.NewHandler(cfg, sessionMgr, signInValidator, createSessionUC)
  ```

  ただし、この変更は他のハンドラー（`popularWorkHandler` 等）の `NewHandler` シグネチャも更新が必要になるため、フェーズ3以降で実コード変更と一緒に更新するほうが自然かもしれない。

  **対応方針**:
  - [x] この PR で更新する
  - [ ] フェーズ 3 以降で実コード変更と合わせて更新する（TODO コメントやメモを残す）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

- **[ドキュメント内の一貫性 - 実装例]**: 「例 2: 複数エンドポイント（password_reset/）」（386行目付近）の Handler 構造体が `repository` パッケージを import し、`queries *repository.Queries` フィールドを持っている。これは新たに追加された「Handler は query, repository への直接アクセス禁止」というルールと矛盾する。

  ```go
  // 現在の例（394行目付近）
  import (
      repository "github.com/mewstcom/mewst/internal/repository/sqlc"
  )

  type Handler struct {
      queries     *repository.Queries
  }
  ```

  **修正案**:

  ルーティング例と同様、フェーズ 3 以降で UseCase を使うパターンに更新するか、この PR で先行して更新する。

  **対応方針**:
  - [x] この PR で更新する
  - [ ] フェーズ 3 以降で実コード変更と合わせて更新する
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

タスク 1-1 で定義された全 7 項目の要件がすべて適切に反映されている。変更は4つのドキュメント（CLAUDE.md、architecture-guide.md、handler-guide.md、validation-guide.md）に一貫して適用されており、以下の点が良い:

- **ルールの明確化**: 「Handler は Repository に直接依存しない」というルールが CLAUDE.md、architecture-guide.md の両方に明記され、依存関係の図解も更新されている
- **UseCase の役割拡張**: 書き込み/読み取りの2種類に分類するテーブルと、具体的なコード例（`GetActiveEmailConfirmationUsecase`）が追加されている
- **Validator 分離の一貫性**: handler-guide.md、validation-guide.md、CLAUDE.md のすべてで命名規則（`{Handler}{Action}Validator`）やパッケージ配置（`internal/validator/`）が統一されている
- **不要な記述の削除**: handler-guide.md から `validator.go` のインライン実装例が削除され、validation-guide.md への参照に置き換えられている

指摘した 3 件はいずれも「ドキュメント内の既存コード例が新ルールと矛盾する」という一貫性の問題であり、この PR で対応するかフェーズ 3 以降で実コード変更と合わせて更新するかは判断が必要。機能的な影響はない。
