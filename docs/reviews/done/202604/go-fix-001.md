# コードレビュー: go-fix

## レビュー情報

| 項目                       | 内容                                 |
| -------------------------- | ------------------------------------ |
| レビュー日                 | 2026-04-03                           |
| 対象ブランチ               | go-fix                               |
| ベースブランチ             | go                                   |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md        |
| 変更ファイル数             | 26 ファイル                          |
| 変更行数（実装）           | +339 / -181 行（自動生成ファイル含む） |
| 変更行数（テスト）         | なし                                 |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

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
- [ ] `go/internal/templates/components/flash.templ`
- [x] `go/internal/templates/layouts/simple.templ`
- [ ] `go/internal/templates/pages/email_confirmation/new.templ`
- [ ] `go/internal/templates/pages/password/edit.templ`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/web/style.css`

### 自動生成ファイル

- [x] `go/internal/templates/components/flash_templ.go`
- [x] `go/internal/templates/layouts/simple_templ.go`
- [x] `go/internal/templates/pages/email_confirmation/new_templ.go`
- [x] `go/internal/templates/pages/password/edit_templ.go`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`

### 設定・その他

- [x] `go/internal/i18n/locales/en.toml`
- [x] `go/internal/i18n/locales/ja.toml`
- [x] `docs/plans/1_doing/sign-up.md`

## ファイルごとのレビュー結果

### `go/internal/templates/components/flash.templ`: flashIcon の CSS クラス命名が不統一

**ステータス**: 要修正

**チェックしたガイドライン**:

- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - テンプレートガイド
- 既存コードとの一貫性

**問題点・改善提案**:

- **一貫性**: `flashIcon` 内で、success/error/warning は `fill-(--success)` 等の CSS 変数参照構文を使用しているが、info のみ `fill-info`（Tailwind ユーティリティクラス）を使用しており不統一

  ```templ
  // 現在のコード（不統一）
  case session.FlashSuccess:
      @templates.Icon("success", "fill-(--success)")
  case session.FlashError:
      @templates.Icon("error", "fill-(--error)")
  case session.FlashWarning:
      @templates.Icon("warning", "fill-(--warning)")
  default:
      @templates.Icon("info", "fill-info")
  ```

  **修正案**: `style.css` で `--color-info`, `--color-success` 等の Tailwind 向けマッピングが追加されているため、すべて Tailwind ユーティリティクラスに統一する

  ```templ
  case session.FlashSuccess:
      @templates.Icon("success", "fill-success")
  case session.FlashError:
      @templates.Icon("error", "fill-error")
  case session.FlashWarning:
      @templates.Icon("warning", "fill-warning")
  default:
      @templates.Icon("info", "fill-info")
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] 全て Tailwind ユーティリティクラス（`fill-success`, `fill-error` 等）に統一する
  - [ ] 全て CSS 変数参照構文（`fill-(--info)` 等）に統一する
  - [ ] 現状維持（動作に支障なし）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/email_confirmation/new.templ`: CSS クラス名が部分的に旧スタイルのまま

**ステータス**: 要修正

**チェックしたガイドライン**:

- 既存コードとの一貫性

**問題点・改善提案**:

- **一貫性**: この PR で `bg-(--primary)` を `bg-primary` に変更しているが、同じファイル内の `text-(--error)`（73行目）は更新されていない。`sign_up/new.templ` では `text-error` に統一済み

  ```templ
  // 73行目: 旧スタイルのまま
  <p class="text-sm text-(--error)">
  ```

  **修正案**:

  ```templ
  <p class="text-sm text-error">
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] `text-error` に変更する
  - [ ] 現状維持（別 PR で対応）
  - [ ] その他（下の回答欄に記入）

  **回答**:

  ```
  （ここに回答を記入）
  ```

### `go/internal/templates/pages/password/edit.templ`: CSS クラス名が旧スタイルのまま

**ステータス**: 要修正

**チェックしたガイドライン**:

- 既存コードとの一貫性

**問題点・改善提案**:

- **一貫性**: このファイルは Flash フィールドの削除のみ変更されているが、他のテンプレートで更新された CSS クラス名が旧スタイルのまま残っている

  1. 20行目: `bg-(--primary)` → 他ファイルでは `bg-primary` に変更済み
  2. 73行目: `text-(--error)` → `sign_up/new.templ` では `text-error` に変更済み

  ```templ
  // 20行目
  <div class="bg-(--primary) rounded-xl">

  // 73行目
  <p class="text-sm text-(--error)">
  ```

  **修正案**:

  ```templ
  // 20行目
  <div class="bg-primary rounded-xl">

  // 73行目
  <p class="text-sm text-error">
  ```

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [x] この PR で `bg-primary` と `text-error` に統一する
  - [ ] 現状維持（別 PR で対応）
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

フラッシュメッセージのページレベルからレイアウトレベルへの移動、および DaisyUI alert から basecoat-css toast パターンへの変更は適切に実装されています。`SimpleLayoutData` 構造体の導入は templ ガイドラインの「構造体ベースのパターン」に合致しており、良い変更です。

全ハンドラーで `layouts.Simple` の呼び出しが一貫して更新されており、i18n キーの追加・命名変更も適切です。

指摘事項は CSS クラス名の統一に関するもの（3件）であり、動作への影響はありませんが、コードベースの一貫性のために対応を推奨します。
