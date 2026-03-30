# コードレビュー: sign-up-2-1

## レビュー情報

| 項目               | 内容        |
| ------------------ | ----------- |
| レビュー日         | 2026-02-04  |
| 対象ブランチ       | sign-up-2-1 |
| ベースブランチ     | sign-up     |
| 変更ファイル数     | 11 ファイル |
| 変更行数（実装）   | +524 行     |
| 変更行数（テスト） | +660 行     |

## 参照するガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - コーディング規約、コミットメッセージ、コメントのガイドライン
- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - Go 版固有のガイドライン
  - HTTPハンドラー
  - バリデーション
  - 国際化（I18n）
  - テスト戦略
  - セキュリティガイドライン

## 変更ファイル一覧

### 実装ファイル

- [x] `go/internal/handler/sign_up/handler.go`
- [x] `go/internal/handler/sign_up/new.go`
- [x] `go/internal/handler/sign_up/create.go`
- [x] `go/internal/handler/sign_up/validator.go`
- [x] `go/internal/templates/pages/sign_up/new.templ`
- [x] `go/internal/templates/pages/sign_up/new_templ.go`（自動生成）

### テストファイル

- [x] `go/internal/handler/sign_up/handler_test.go`
- [x] `go/internal/handler/sign_up/validator_test.go`

### 設定・その他

- [x] `go/internal/i18n/locales/ja.toml`
- [x] `go/internal/i18n/locales/en.toml`
- [x] `docs/designs/1_doing/sign-up.md`

## ファイルごとのレビュー結果

### `go/internal/handler/sign_up/handler.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#HTTPハンドラー](/workspace/go/CLAUDE.md) - handler.go の構造
- [@go/CLAUDE.md#アーキテクチャ](/workspace/go/CLAUDE.md) - 依存関係のルール

**問題点・改善提案**:

- 問題なし
- Handler 構造体が必要な依存関係を適切に受け取っている
- `NewHandler` でバリデーターを内部で生成するパターンも問題なし

### `go/internal/handler/sign_up/new.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#HTTPハンドラー](/workspace/go/CLAUDE.md) - new.go のメソッド名
- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - CSRF対策

**問題点・改善提案**:

- 問題なし
- CSRFトークンの取得、テンプレートへの受け渡しが適切
- ロケール設定、ページメタ情報の設定も既存パターンに準拠

### `go/internal/handler/sign_up/create.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#HTTPハンドラー](/workspace/go/CLAUDE.md) - create.go のメソッド名
- [@go/CLAUDE.md#ログ出力](/workspace/go/CLAUDE.md) - slog の使用
- [@go/CLAUDE.md#セキュリティガイドライン](/workspace/go/CLAUDE.md) - CSRF対策、Bot対策

**問題点・改善提案**:

- 問題なし
- レート制限、Turnstile検証、バリデーション、ユースケース呼び出しの順序が適切
- `slog.ErrorContext` / `slog.WarnContext` の使用も正しい
- エラー時のステータスコード（422 Unprocessable Entity）も適切

### `go/internal/handler/sign_up/validator.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#リクエストバリデーション](/workspace/go/CLAUDE.md) - バリデーターの構造
- [@go/docs/validation-guide.md](/workspace/go/docs/validation-guide.md) - バリデーションパターン

**問題点・改善提案**:

- 問題なし
- 形式チェック（必須、メール形式）と状態チェック（DB検証）を統合
- `net/mail.ParseAddress` でのメール形式チェックも標準的な方法
- エラー時に早期リターンする設計も適切

### `go/internal/templates/pages/sign_up/new.templ`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#templテンプレート](/workspace/go/CLAUDE.md) - 構造体ベースの引数パターン
- [@go/CLAUDE.md#国際化](/workspace/go/CLAUDE.md) - templates.T() の使用

**問題点・改善提案**:

- 問題なし
- `NewPageData` 構造体を使用した引数パターンが正しい
- CSRFトークン、Turnstile コンポーネントの配置が適切
- 国際化対応（`templates.T(ctx, "...")` の使用）も正しい

### `go/internal/handler/sign_up/handler_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - 実データベーステスト
- [@go/CLAUDE.md#テストのベストプラクティス](/workspace/go/CLAUDE.md) - 並行テスト、テストヘルパー

**問題点・改善提案**:

- 問題なし
- `testutil.SetupTestDB(t)` を使用した実データベーステスト
- `t.Parallel()` による並行テスト
- 正常系と異常系（空メール、不正メール、Turnstile失敗、メール重複）を網羅

### `go/internal/handler/sign_up/validator_test.go`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#テスト戦略](/workspace/go/CLAUDE.md) - バリデーターのテスト

**問題点・改善提案**:

- 問題なし
- 空メール、不正形式、有効なメール、重複メール、大文字小文字の区別なしのケースを網羅
- テストデータの作成に `testutil.NewUserBuilder` を使用

### `go/internal/i18n/locales/ja.toml`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#国際化](/workspace/go/CLAUDE.md) - 翻訳ファイルの形式

**問題点・改善提案**:

- 問題なし
- `description` と `other` のフォーマットに準拠
- キー名が既存パターンと一貫性あり

### `go/internal/i18n/locales/en.toml`

**ステータス**: OK

**チェックしたガイドライン**:

- [@go/CLAUDE.md#国際化](/workspace/go/CLAUDE.md) - 翻訳ファイルの形式

**問題点・改善提案**:

- 問題なし
- 日本語版と同じキーを定義

## 総合評価

**評価**: Approve

**総評**:

サインアップハンドラーの実装は、プロジェクトのガイドラインに準拠しており、高品質なコードです。

**良かった点**:

1. **セキュリティ対策が適切**: CSRF対策、Turnstile（Bot対策）、レート制限が正しく実装されている
2. **アーキテクチャの一貫性**: 既存の `password_reset` ハンドラーと同様の構造を採用
3. **テストの網羅性**: 正常系と異常系を幅広くテスト（空メール、不正形式、Turnstile失敗、メール重複など）
4. **国際化対応**: すべてのユーザー向けメッセージが翻訳ファイルに定義されている
5. **ログ出力**: `slog` パッケージを使用した構造化ログ

**確認事項**:

- 特になし。このPRはマージ可能です。

---

## 質問と回答

質問はありません。
