# コードレビュー: go-fix

## レビュー情報

| 項目                       | 内容                                   |
| -------------------------- | -------------------------------------- |
| レビュー日                 | 2026-04-03                             |
| 対象ブランチ               | go-fix                                 |
| ベースブランチ             | go                                     |
| 作業計画書（指定があれば） | docs/plans/1_doing/sign-up.md          |
| 変更ファイル数             | 35 ファイル                            |
| 変更行数（実装）           | +856 / -213 行（自動生成ファイル含む） |
| 変更行数（テスト）         | なし                                   |

## 参照するガイドライン

- [@CLAUDE.md#レビュー時に参照するガイドライン](/workspace/CLAUDE.md) - ガイドライン一覧
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go版の開発ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templテンプレートガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 過去のレビューからの対応状況

前回レビュー（`go-fix-001.md`, `go-fix-002.md`, `go-fix-003.md`）で対応方針が決定済みの問題（5件）をすべて確認し、全件対応済みであることを確認しました：

1. **flash.templ CSS クラス命名不統一** → `fill-success`, `fill-error` 等の Tailwind ユーティリティクラスに統一済み ✓
2. **email_confirmation/new.templ `text-(--error)`** → `text-error` に変更済み ✓
3. **password/edit.templ `bg-(--primary)`, `text-(--error)`** → `bg-primary`, `text-error` に変更済み ✓
4. **sign_up/new.templ ボタン CSS 不統一** → 開発者の回答に基づき全ページを `btn-primary rounded-full w-fit` に統一済み ✓
5. **password_reset/new.templ, sign_in/new.templ, accounts/new.templ CSS クラス旧スタイル** → `bg-primary`, `text-error` に変更済み ✓

**追加確認**: `grep -rn "bg-(--\|text-(--" templates/pages/ --include="*.templ"` で旧スタイルの CSS クラスが残存していないことを確認済み。

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
- [x] `go/internal/templates/pages/password_reset/new.templ`
- [x] `go/internal/templates/pages/sign_in/new.templ`
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
- [x] `docs/reviews/go-fix-003.md`

## ファイルごとのレビュー結果

すべてのファイルで問題は検出されませんでした。各ファイルのチェック結果は以下の通りです：

**ハンドラー（12ファイル）**: 全ハンドラーで `layouts.Simple(layouts.SimpleLayoutData{...}, content)` の呼び出しパターンが統一されている。フラッシュメッセージを表示するページ（`New` ハンドラー）では `Flash: flash` を渡し、フォーム再表示時（`renderForm`）では `Flash` を省略するパターンが一貫している。

**テンプレート（8ファイル）**: CSS クラス名が `bg-primary`, `text-error`, `btn-primary rounded-full w-fit`, `fill-success` 等の Tailwind ユーティリティクラスに統一されている。`flash.templ` の toast パターン実装にはアクセシビリティ属性（`role="status"`, `aria-atomic="true"`, `aria-hidden="false"`）が含まれている。

**i18n（2ファイル）**: `flash_dismiss` キーの追加、`already_have_account` → `sign_up_have_account` + `sign_up_sign_in` への分割、各キーの命名が `{page_name}_{detail}` の規則に従っている。

**CSS（1ファイル）**: `--color-error`, `--color-success` 等の Tailwind v4 向けカラー変数マッピングの追加が適切。

## 設計改善の提案

設計改善の提案はありません。

## 総合評価

**評価**: Approve

**総評**:

過去3回のレビュー（go-fix-001〜003）で指摘された問題（計5件）がすべて対応済みであり、旧スタイルの CSS クラス名の残存もないことを確認しました。

本 PR の主な変更は以下のとおりで、すべて適切に実装されています：

- **フラッシュメッセージのレイアウトレベル化**: `SimpleLayoutData` 構造体の導入により、ページテンプレートから `Flash` フィールドが除去され、レイアウトに集約。templ ガイドの「構造体ベースのパターン」に合致
- **basecoat-css toast パターンへの移行**: アクセシビリティ属性、`flash_dismiss` i18n キー、タイプ別アイコン表示、閉じるボタンが作業計画書（Phase 7-1）の要件どおりに実装
- **CSS クラス名の統一**: 全ページテンプレートで `bg-primary`, `text-error`, `btn-primary`, `fill-success` 等の Tailwind ユーティリティクラスに統一
- **i18n キーの整理**: `already_have_account` → `sign_up_have_account` + `sign_up_sign_in` への分割により翻訳の柔軟性が向上
- **全ハンドラーの統一的な更新**: `layouts.Simple` の呼び出しが一貫したパターンで更新

コードベースの一貫性が保たれており、マージ可能な状態です。
