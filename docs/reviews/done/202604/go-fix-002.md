# コードレビュー: go-fix

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-04-03                             |
| 対象ブランチ               | go-fix                                 |
| ベースブランチ             | go                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md          |
| 変更ファイル数             | 27 ファイル                            |
| 変更行数（実装）           | +555 / -187 行（自動生成ファイル含む） |
| 変更行数（テスト）         | なし                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド

## 過去のレビューからの対応状況

前回レビュー（`go-fix-001.md`）で対応方針が決定済みの問題（3件）をすべて確認し、全件対応済みであることを確認しました：

1. **flash.templ CSS クラス命名不統一** → `fill-success`, `fill-error` 等の Tailwind ユーティリティクラスに統一済み ✓
2. **email_confirmation/new.templ `text-(--error)`** → `text-error` に変更済み ✓
3. **password/edit.templ `bg-(--primary)`, `text-(--error)`** → `bg-primary`, `text-error` に変更済み ✓

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
- [x] `go/internal/templates/pages/email_confirmation/new.templ`
- [x] `go/internal/templates/pages/password/edit.templ`
- [ ] `go/internal/templates/pages/sign_up/new.templ`
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
- [x] `docs/reviews/go-fix-001.md`

## ファイルごとのレビュー結果

### `go/internal/templates/pages/sign_up/new.templ`: ボタンの CSS クラスが他ページと不統一

**ステータス**: 要確認

**チェックしたガイドライン**:

- [@CLAUDE.md#既存コードとの一貫性](/workspace/CLAUDE.md) - 実装時のガイドライン

**問題点・改善提案**:

- **一貫性**: `sign_up/new.templ` のサブミットボタンが `btn-primary rounded-full w-fit` を使用しているが、他のすべてのページテンプレートでは `btn rounded-full w-fit text-black` を使用しており、不統一

  ```
  # 他の全ページ（sign_in, password_reset, email_confirmation, password, accounts）
  class="btn rounded-full w-fit text-black"

  # sign_up/new.templ のみ
  class="btn-primary rounded-full w-fit"
  ```

  **修正案A**: 他ページと合わせて `btn rounded-full w-fit text-black` に変更する

  ```templ
  <button
      class="btn rounded-full w-fit text-black"
      tabindex="2"
      type="submit"
  >
  ```

  **修正案B**: これが新しいデザインパターンであれば、他ページも段階的に `btn-primary` に移行する

  **対応方針**:

  <!-- 開発者が回答を記入してください -->
  - [ ] 案A: `btn rounded-full w-fit text-black` に統一する
  - [ ] 案B: 意図的な変更であり、他ページも今後 `btn-primary` に移行予定
  - [x] その他（下の回答欄に記入）

  **回答**:

  ```
  他ページも `btn-primary` に移行するように修正をお願いします。
  ```

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Comment

**総評**:

前回レビュー（go-fix-001）で指摘された CSS クラス名の不統一（3件）がすべて修正されており、対応品質は良好です。

本 PR の主な変更（フラッシュメッセージのページレベル → レイアウトレベル移動、basecoat-css toast パターンへの変更）は適切に実装されています。具体的には：

- `SimpleLayoutData` 構造体の導入は templ ガイドラインの「構造体ベースのパターン」に合致
- ページテンプレートからの `Flash` フィールド削除とレイアウトへの集約が一貫して適用されている
- 全ハンドラーで `layouts.Simple` の呼び出しが統一的に更新されている
- `flash.templ` の toast パターン実装にはアクセシビリティ属性（`role`, `aria-atomic`, `aria-hidden`）が含まれている
- i18n キーの追加・リネームが `{page_name}_{detail}` の命名規則に従っている
- CSS カラー変数のマッピング追加により、Tailwind ユーティリティクラスでの参照が可能になっている

指摘事項は sign_up ページのボタンクラスの不統一（1件）のみであり、動作への影響はありません。
