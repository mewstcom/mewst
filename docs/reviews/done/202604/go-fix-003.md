# コードレビュー: go-fix

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-04-03                             |
| 対象ブランチ               | go-fix                                 |
| ベースブランチ             | go                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md          |
| 変更ファイル数             | 34 ファイル                            |
| 変更行数（実装）           | +702 / -197 行（自動生成ファイル含む） |
| 変更行数（テスト）         | なし                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 過去のレビューからの対応状況

前回レビュー（`go-fix-001.md`, `go-fix-002.md`）で対応方針が決定済みの問題（4件）をすべて確認し、全件対応済みであることを確認しました：

1. **flash.templ CSS クラス命名不統一** → `fill-success`, `fill-error` 等の Tailwind ユーティリティクラスに統一済み ✓
2. **email_confirmation/new.templ `text-(--error)`** → `text-error` に変更済み ✓
3. **password/edit.templ `bg-(--primary)`, `text-(--error)`** → `bg-primary`, `text-error` に変更済み ✓
4. **sign_up/new.templ ボタン CSS 不統一** → 開発者の回答に基づき全ページを `btn-primary rounded-full w-fit` に統一済み ✓

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/accounts/create.go`
- [x] `go/internal/handler/accounts/new.go`
- [x] `go/internal/handler/email_confirmation/create.go`
- [x] `go/internal/handler/email_confirmation/new.go`
- [x] `go/internal/handler/password/edit.go`
- [x] `go/internal/handler/password/update.go`
- [x] `go/internal/handler/password_reset/create.go`
- [x] `go/internal/handler/password_reset/new.go`
- [x] `go/internal/handler/sign_in/create.go`
- [x] `go/internal/handler/sign_in/new.go`
- [x] `go/internal/handler/sign_up/create.go`
- [x] `go/internal/handler/sign_up/new.go`
- [x] `go/internal/templates/components/flash.templ`
- [x] `go/internal/templates/layouts/simple.templ`
- [x] `go/internal/templates/pages/accounts/new.templ`
- [x] `go/internal/templates/pages/email_confirmation/new.templ`
- [x] `go/internal/templates/pages/password/edit.templ`
- [ ] `go/internal/templates/pages/password_reset/new.templ`
- [ ] `go/internal/templates/pages/sign_in/new.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/web/style.css`

### 自動生成ファイル

- [x] `go/internal/templates/components/flash_templ.go`
- [x] `go/internal/templates/layouts/simple_templ.go`
- [x] `go/internal/templates/pages/accounts/new_templ.go`
- [x] `go/internal/templates/pages/email_confirmation/new_templ.go`
- [x] `go/internal/templates/pages/password/edit_templ.go`
- [x] `go/internal/templates/pages/password_reset/new_templ.go`
- [x] `go/internal/templates/pages/sign_in/new_templ.go`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`

### 設定・その他

- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `docs/plans/1_doing/sign-up.md`
- [x] `docs/reviews/go-fix-001.md`
- [x] `docs/reviews/go-fix-002.md`

## ファイルごとのレビュー結果

### `go/internal/templates/pages/password_reset/new.templ`, `go/internal/templates/pages/sign_in/new.templ`, `go/internal/templates/pages/accounts/new.templ`: CSS クラス名が旧スタイルのまま

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@CLAUDE.md#既存コードとの一貫性](/workspace/CLAUDE.md) - 実装時のガイドライン

**問題点・改善提案**:

- **一貫性**: この PR で `email_confirmation/new.templ`、`password/edit.templ`、`sign_up/new.templ` は `bg-primary` / `text-error` に更新済みだが、以下の3ファイルは旧スタイルのまま残っている

  **password_reset/new.templ**:
  - 22行目: `bg-(--primary)` → `bg-primary`
  - 70行目: `text-(--error)` → `text-error`

  **sign_in/new.templ**:
  - 23行目: `bg-(--primary)` → `bg-primary`
  - 71行目, 105行目: `text-(--error)` → `text-error`

  **accounts/new.templ**:
  - 23行目: `bg-(--primary)` → `bg-primary`
  - 88行目, 120行目: `text-(--error)` → `text-error`

  **修正案**: 上記すべてのファイルで `bg-(--primary)` → `bg-primary`、`text-(--error)` → `text-error` に変更し、`make templ-generate` で再生成する

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] この PR で3ファイルすべてを更新する
  - [ ] 別 PR で対応する
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

前回レビュー（go-fix-001, go-fix-002）で指摘された問題（4件）はすべて対応済みです。

本 PR の主な変更は以下のとおりで、いずれも適切に実装されています：

- **フラッシュメッセージのレイアウトレベル化**: `SimpleLayoutData` 構造体の導入により、ページテンプレートから `Flash` フィールドが除去され、レイアウトに集約。templ ガイドの「構造体ベースのパターン」に合致
- **basecoat-css toast パターンへの移行**: アクセシビリティ属性（`role`, `aria-atomic`, `aria-hidden`）、`flash_dismiss` i18n キー、タイプ別アイコン表示が適切に実装
- **全ハンドラーの統一的な更新**: `layouts.Simple` の呼び出しが一貫してパターン化（フラッシュあり: `SimpleLayoutData{Meta: meta, Flash: flash}`、なし: `SimpleLayoutData{Meta: meta}`）
- **ボタンクラスの全ページ統一**: go-fix-002 での開発者回答に基づき全ページを `btn-primary rounded-full w-fit` に更新
- **i18n キーの整理**: `already_have_account` → `sign_up_have_account` + `sign_up_sign_in` への分割は翻訳柔軟性の向上として適切

指摘事項は CSS クラス名の旧スタイル残留（3ファイル）のみであり、動作への影響はありませんが、前回・前々回のレビューで同種の修正を行った一貫性の観点から対応を推奨します。
