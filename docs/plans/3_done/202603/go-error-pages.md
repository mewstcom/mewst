# Go 版エラーページの実装 作業計画書

## 仕様書

- タスク完了後に作成予定: `docs/specs/error-page/overview.md`

## 概要

Go 版で chi ルーターにマッチしないパスにアクセスした際、Go 標準の素のテキスト（`"404 page not found"`）が表示される。Rails 版には `public/404.html` としてスタイル付きの 404 ページが存在するため、Go 版にも同等のエラーページを実装する。

また、リバースプロキシの 502 エラーページが `render502ErrorHTML()` 関数内にハードコードされた HTML 文字列で実装されているため、これも templ テンプレートに移行してエラー系テンプレートを一箇所に集約する。

ユーザーが存在しないページにアクセスした際や、Rails 版への接続エラーが発生した際に、アプリケーションのデザインに沿ったエラーページを表示することで、ユーザー体験を改善する。

## 要件

### 機能要件

- 404 レスポンス時にスタイル付きのエラーページが表示される
- 502 レスポンス時にスタイル付きのエラーページが表示される（既存の 502 ページを templ テンプレート化）
- エラーページには「ホームに戻る」リンクが含まれる
- 日本語・英語の両方に対応する（i18n）
- Rails 版のエラーページ（`rails/public/404.html`, `rails/public/500.html`）と同等のデザイン・トーンを維持する

### 非機能要件

- **パフォーマンス**: エラーページの表示はセッション取得や DB アクセスを必要としない（スタンドアロン方式）
- **保守性**: 各ハンドラーの変更を最小限に抑える。エラーページ系テンプレートを一箇所に集約する

## 実装ガイドラインの参照

### Go 版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTP ハンドラーガイドライン
- [@go/docs/i18n-guide.md](/workspace/go/docs/i18n-guide.md) - 国際化ガイド
- [@go/docs/templ-guide.md](/workspace/go/docs/templ-guide.md) - templ テンプレートガイド

## 設計

### 方針: chi の NotFound ハンドラー + 共通ヘルパー関数

chi ルーターには `r.NotFound()` メソッドがあり、ルーティングにマッチしなかった場合のハンドラーを設定できる。これに加えて、将来的に各ハンドラー内から呼び出せる共通のヘルパー関数を用意する。

### エラーページのテンプレート

レイアウトに依存しないスタンドアロンの templ コンポーネントとして実装する。

理由:

- エラー発生時にはセッションや DB が利用できない可能性がある
- ログインユーザー情報の取得が不要で、シンプルに保てる
- 各エラーページで統一したパターンを使用できる

### ファイル構成

```
go/internal/templates/pages/errors/
├── not_found.templ          # 404エラーページテンプレート
└── bad_gateway.templ        # 502エラーページテンプレート

go/internal/handler/
└── errors.go                # NotFound ヘルパー関数（全ハンドラーで共用）
```

### テンプレート設計

Rails 版の 404 ページ（`rails/public/404.html`）を参考に、以下の要素を含める：

- 絵文字（🫥）による視覚的なインジケーター（404 は Rails 版と同じ 🫥 を使用）
- エラーメッセージ（i18n 対応）
- 「ホームに戻る」リンク
- CSS はインライン埋め込み（外部依存なし）
- Rails 版の 404 ページ（`rails/public/404.html`）のスタイルを踏襲

502 ページも同様のデザインで統一する：

- 絵文字（⚠️）による視覚的なインジケーター（既存の 502 ページと同じ）
- エラーメッセージ（i18n 対応）
- 「ホームに戻る」リンク

### i18n 翻訳キー

```toml
# ja.toml
[error_not_found_message]
description = "404エラーページのメッセージ"
other = "お探しのページは見つかりませんでした"

[error_not_found_back_to_home]
description = "404エラーページのホームリンク"
other = "ホームに戻る"

[error_bad_gateway_message]
description = "502エラーページのメッセージ"
other = "申し訳ございません。現在サービスに接続できません。"

[error_bad_gateway_submessage]
description = "502エラーページのサブメッセージ"
other = "しばらくしてから再度お試しください。"

[error_bad_gateway_back_to_home]
description = "502エラーページのホームリンク"
other = "ホームに戻る"

# en.toml
[error_not_found_message]
description = "404 error page message"
other = "The page you are looking for could not be found"

[error_not_found_back_to_home]
description = "404 error page link to home"
other = "Back to home"

[error_bad_gateway_message]
description = "502 error page message"
other = "Sorry, we are unable to connect to the service."

[error_bad_gateway_submessage]
description = "502 error page submessage"
other = "Please try again later."

[error_bad_gateway_back_to_home]
description = "502 error page link to home"
other = "Back to home"
```

### ヘルパー関数

```go
// go/internal/handler/errors.go
package handler

// NotFound はスタイル付きの404ページをレンダリングする
func NotFound(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    errors.NotFound().Render(r.Context(), w)
}
```

### chi ルーターの設定

`cmd/server/main.go` で chi ルーターに NotFound ハンドラーを登録する。これにより、ルーティングにマッチしなかった場合も同じエラーページが表示される。

```go
r.NotFound(handler.NotFound)
```

### 502 ページのリファクタリング

`internal/middleware/reverse_proxy.go` の `render502ErrorHTML()` 関数を削除し、代わりに `errors.BadGateway()` templ コンポーネントを使用する。

```go
// 変更前
w.Write([]byte(render502ErrorHTML()))

// 変更後
errors.BadGateway().Render(r.Context(), w)
```

## 採用しなかった方針

### A. デフォルトレイアウト内でエラーページを表示する

サイドバーやナビゲーションを含むデフォルトレイアウトを使って 404 ページを表示する方針。

**不採用の理由**:

- 404 発生時にセッション情報やサイドバーデータの取得が必要になり、追加の DB アクセスが発生する
- セッションや DB が利用できない場合にエラーページ自体が表示できなくなるリスクがある

### B. `http.NotFound` をそのまま使い、ミドルウェアでレスポンスを書き換える

`ResponseWriter` をラップするミドルウェアを作成し、ステータスコード 404 のレスポンスをインターセプトしてエラーページに差し替える方針。

**不採用の理由**:

- `ResponseWriter` のラップは実装が複雑になる（`WriteHeader` と `Write` の順序制御、`Flush` や `Hijack` インターフェースの対応など）
- 各ハンドラーのヘルパー関数置き換えの方がシンプルで明示的

### C. 502 ページは既存のハードコード HTML のまま残す

`render502ErrorHTML()` を変更せず、404 のみ templ テンプレート化する方針。

**不採用の理由**:

- エラーページ系のテンプレートが分散する（502 は Go 文字列リテラル、404 は templ）
- i18n 対応が困難（既存の 502 ページは日本語のみ）
- templ テンプレートに統一することで保守性が向上する

## タスクリスト

### フェーズ 1: 404 エラーページの実装

- [x] **1-1**: [Go] 404 エラーページテンプレートと共通ヘルパーの実装
  - `go/internal/templates/pages/errors/not_found.templ` を作成（Rails 版 `public/404.html` のデザインを踏襲）
  - `go/internal/handler/errors.go` に `NotFound` ヘルパー関数を作成
  - i18n 翻訳キーを `ja.toml`、`en.toml` に追加
  - `cmd/server/main.go` で chi ルーターに `r.NotFound()` を設定
  - テスト: `NotFound` ヘルパー関数のテスト、テンプレートレンダリングのテスト
  - **想定ファイル数**: 約 6 ファイル（実装 4 + テスト 2）
  - **想定行数**: 約 180 行（実装 130 行 + テスト 50 行）

### フェーズ 2: 502 エラーページの templ テンプレート化

- [x] **2-1**: [Go] 502 エラーページを templ テンプレートに移行
  - `go/internal/templates/pages/errors/bad_gateway.templ` を作成
  - `go/internal/middleware/reverse_proxy.go` の `render502ErrorHTML()` を削除し、templ テンプレートを使用するように変更
  - i18n 翻訳キーを `ja.toml`、`en.toml` に追加
  - テスト: テンプレートレンダリングのテスト
  - **想定ファイル数**: 約 5 ファイル（実装 4 + テスト 1）
  - **想定行数**: 約 120 行（実装 90 行 + テスト 30 行）

### フェーズ 3: 仕様書への反映

- [x] **3-1**: 仕様書の作成・更新
  - `docs/specs/error-page/overview.md` に仕様書を作成する
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **500 エラーページ**: 現在 `http.Error(w, "Internal Server Error", 500)` で返している 28 箇所の改善は別途計画する。500 ページはエラーの種類やリカバリー方法が多様であり、単純なテンプレート差し替えでは対応できないため
- **chi Recoverer のカスタマイズ**: パニック時のエラーページのカスタマイズは 500 エラーページの計画と合わせて行う
- **403 Forbidden ページ**: 現在は CSRF エラー時にのみ使用しており、専用のスタイル付きページは不要

## 参考資料

- [Wikino Go 版エラーページの実装 作業計画書](/wikino/docs/plans/3_done/202603/go-error-pages.md)
- [Rails 版 404 ページ](/workspace/rails/public/404.html)
- [Rails 版 500 ページ](/workspace/rails/public/500.html)
