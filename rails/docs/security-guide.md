# セキュリティガイドライン

このドキュメントは、Rails 版 Mewst のセキュリティガイドラインを説明します。

## 基本方針

Web アプリケーションのセキュリティは**最優先事項**です。

## 基本対策

- **CSRF 対策**: `protect_from_forgery` がデフォルトで有効、`form_with` ヘルパーを使用
- **XSS 対策**: ERB の自動エスケープを活用、`raw`/`html_safe` は慎重に使用
- **SQL インジェクション対策**: ActiveRecord のプリペアドステートメント、プレースホルダーを使用
- **認証**: bcrypt（`has_secure_password`）で管理
- **Strong Parameters**: すべてのコントローラーで使用
