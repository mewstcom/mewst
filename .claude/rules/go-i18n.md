---
paths:
  - "go/**/*.{go,templ}"
---

# 国際化（I18n）ガイド

このドキュメントは、Go 版プロジェクトでの国際化（Internationalization）のベストプラクティスを説明します。

## 概要

すべてのユーザー向けメッセージは**必ず国際化対応**します。

### 対応言語

- **日本語**（デフォルト）
- **英語**

### 翻訳ファイル

- `internal/i18n/locales/ja.toml` - 日本語翻訳
- `internal/i18n/locales/en.toml` - 英語翻訳

## 使用方法

### テンプレートでの使用

```templ
// pages/sign_in.templ
package pages

import (
    "context"
    "example.com/app/internal/templates"
)

templ New(ctx context.Context, csrfToken string) {
    <div>
        // ページ固有テキスト (page_name = sign_in_new)
        <h2>{ templates.T(ctx, "sign_in_new_title") }</h2>

        <label for="email">
            { templates.T(ctx, "sign_in_new_email_label") }
        </label>

        <button type="submit">{ templates.T(ctx, "sign_in_new_submit") }</button>
    </div>
}
```

### Goコードでの使用

```go
// internal/handler/sign_in.go
package handler

import (
    "example.com/app/internal/i18n"
)

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // フラッシュ共通メッセージ (flash_* 名前空間)
    h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_sign_in_success"))

    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

### プレースホルダー付き翻訳

```templ
// テンプレート
<p>{ templates.T(ctx, "watchers_count", map[string]any{"Count": work.WatchersCount}) }</p>
```

```toml
# ja.toml
[watchers_count]
description = "ウォッチ数の表示"
other = "{{.Count}} 人がウォッチ"

# en.toml
[watchers_count]
description = "Display watchers count"
other = "{{.Count}} watchers"
```

### 条件に応じた翻訳

```templ
templ WorkCard(ctx context.Context, work viewmodel.Work) {
    <div>
        <h3>{ work.Title }</h3>

        // シーズン表示
        if work.SeasonYear != nil && work.SeasonName != nil {
            <p>
                { fmt.Sprintf("%d年", *work.SeasonYear) }
                switch *work.SeasonName {
                    case "spring":
                        { templates.T(ctx, "season_spring") }
                    case "summer":
                        { templates.T(ctx, "season_summer") }
                    case "autumn":
                        { templates.T(ctx, "season_autumn") }
                    case "winter":
                        { templates.T(ctx, "season_winter") }
                }
            </p>
        }
    </div>
}
```

## 対象メッセージ

### 国際化が必要なもの

- ✅ ページタイトル、見出し
- ✅ ラベル、ボタンテキスト
- ✅ エラーメッセージ、成功メッセージ
- ✅ ヘルプテキスト、説明文
- ✅ バリデーションメッセージ

### 国際化が不要なもの

- ❌ ログメッセージ（開発者向け）
- ❌ panic メッセージ（想定外のエラー）
- ❌ 内部エラー（ユーザーに見せないエラー）
- ❌ コメント（コード内のコメント）

## 運用・開発者向けメッセージ

運用・開発者向けのメッセージは**日本語で統一**します（国際化不要）。

### ログメッセージ

```go
// ✅ OK: ログメッセージは日本語でOK（開発者向け）
slog.InfoContext(ctx, "パスワードリセット申請を受け付けました", "user_id", userID)
slog.ErrorContext(ctx, "データベース接続エラー", "error", err)
```

### panicメッセージ

```go
// ✅ OK: panicメッセージは日本語でOK
panic("設定ファイルの読み込みに失敗しました")
```

### 内部エラー

```go
// ✅ OK: 内部エラーは日本語でOK（開発者向け）
return fmt.Errorf("トークンのハッシュ化に失敗: %w", err)
```

## 判断基準

```go
// ❌ NG: ユーザー向けメッセージなのに日本語ハードコード
http.Error(w, "メールアドレスを入力してください", http.StatusBadRequest)

// ✅ OK: ユーザー向けメッセージは国際化 (validation_* 名前空間に集約)
ve.AddField("email", i18n.T(ctx, "validation_required"))

// ✅ OK: ログメッセージは日本語で OK (開発者向け)
slog.InfoContext(ctx, "パスワードリセットトークンを生成しました", "user_id", userID)

// ✅ OK: 内部エラーは日本語で OK (開発者向け)
return fmt.Errorf("セッションの作成に失敗: %w", err)
```

## 翻訳キーの分類

翻訳キーは責務別に 4 つのカテゴリに分けて命名する。「ページ固有のテキストは page_name でスコープし、複数ページから参照される共通メッセージは namespace でスコープする」というハイブリッド方式とする。

| カテゴリ                     | プレフィックス  | 用途                                                                          | 例                                                        |
| ---------------------------- | --------------- | ----------------------------------------------------------------------------- | --------------------------------------------------------- |
| ページ固有テキスト           | `{page_name}_*` | 特定ページに紐づく title / heading / label / button / link text 等            | `sign_in_new_title`, `password_edit_password_label`       |
| バリデーション共通メッセージ | `validation_*`  | 複数ページ・複数 validator から参照される入力検証エラー                       | `validation_required`, `validation_email_invalid`         |
| フラッシュ共通メッセージ     | `flash_*`       | 操作完了時にリダイレクト先で表示する成功 / 失敗メッセージ                     | `flash_sign_in_success`, `flash_password_updated`         |
| 共通 UI 要素                 | (名前空間なし)  | サイト全体のデフォルト・複数ページの非ページ固有 UI・エラーページ・メール件名 | `default_title`, `flash_dismiss`, `error_not_found_title` |

### page_name の決め方

- 原則として「ハンドラー + アクション」(`{handler}_{action}`) で構成する。例: `sign_in_new`, `password_edit`, `email_confirmation_new`
- ハンドラーが将来複数のフォームを持っても破綻せず、画面とキーが 1:1 で対応する
- 同じテキストでもページが異なれば別キーとして展開する。例: `back to sign in` リンクは `password_reset_new_back_to_sign_in` / `email_confirmation_new_back_to_sign_in` のように各ページごとに分割する。同じ文言でも将来ページごとに調整したくなった時に局所的に変更できる
- リンクテキストはリンク元ページ専用キーとして定義し、リンク先ページの title をそのまま流用しない。リンク先ページの title 変更で意図せずリンクテキストが変わることを防ぐ

### namespace の決め方

- **`validation_*`**: validator が `*model.ValidationError` に積むメッセージ、および handler が同じ意味で表示する `ve.AddGlobal(...)` 経由のメッセージはここに集約する。ハンドラーが出す Bot 検証失敗・レート制限超過・システムエラーフォールバック等も「ユーザーに『この入力 / 試行は受け付けない』と伝える」性質なら `validation_*` に入れる
- **`flash_*`**: 完了系のフラッシュメッセージ。命名はメッセージの内容を表す動詞主体 (`flash_password_updated`) にし、必ずしも呼び出し元ハンドラー名と一致させる必要はない (例: `password/update` ハンドラーから設定するメッセージでも `flash_password_updated` で OK)
- **共通 UI 要素 (namespace なし)**: サイト全体のデフォルト (`default_title`, `default_description`)、汎用 UI 部品 (`flash_dismiss`)、メール件名 (`{purpose}_email_subject` 等)、エラーページ (`error_not_found_*`, `error_bad_gateway_*`) などはどのカテゴリにも属さないため namespace なしで定義する

## 翻訳の追加手順

### 1. メッセージ ID を決定

[翻訳キーの分類](#翻訳キーの分類) のいずれに該当するかを判断し、対応するプレフィックス / namespace を選ぶ。

例:

- ページ固有: `password_reset_new_title`, `sign_in_new_email_label`, `account_new_atname_hint`
- バリデーション: `validation_required`, `validation_email_invalid`, `validation_password_too_short`
- フラッシュ: `flash_sign_in_success`, `flash_password_updated`
- 共通 UI: `default_title`, `flash_dismiss`, `error_not_found_title`

### 2. ja.toml に追加

```toml
# internal/i18n/locales/ja.toml

[password_reset_new_title]
description = "パスワードリセットページのタイトル"
other = "パスワードを忘れた？"

[password_reset_new_email_label]
description = "パスワードリセットページのメールアドレスラベル"
other = "メールアドレス"

[validation_required]
description = "必須フィールドのバリデーションエラー"
other = "入力してください"

[validation_email_invalid]
description = "メールアドレス形式のバリデーションエラー"
other = "正しいメールアドレスを入力してください"
```

### 3. en.toml に追加

```toml
# internal/i18n/locales/en.toml

[password_reset_new_title]
description = "Password reset page title"
other = "Forgot your password?"

[password_reset_new_email_label]
description = "Password reset page email label"
other = "Email"

[validation_required]
description = "Required field validation error"
other = "is required"

[validation_email_invalid]
description = "Invalid email format validation error"
other = "is not a valid email address"
```

### 4. テンプレートまたは Go コードで使用

```templ
// テンプレート
<h2>{ templates.T(ctx, "password_reset_new_title") }</h2>
```

```go
// Go コード
ve.AddField("email", i18n.T(ctx, "validation_required"))
```

## 翻訳の命名規則

### ページタイトル

**形式**: `{page_name}_title`

例:

- `password_reset_new_title` - パスワードリセット
- `sign_in_new_title` - ログイン
- `popular_anime_index_title` - 人気アニメ一覧

### 見出し

**形式**: `{page_name}_heading`

例:

- `sign_in_new_heading` - ログイン
- `password_reset_new_heading` - パスワードリセット

### ラベル

**形式**: `{page_name}_{field_name}_label`

例:

- `sign_in_new_email_label` - メールアドレス
- `password_reset_new_email_label` - メールアドレス
- `account_new_atname_label` - アットネーム

### 送信ボタン

**形式**: `{page_name}_submit`

例:

- `password_reset_new_submit` - 送信
- `sign_in_new_submit` - ログイン

### ヒント・補足テキスト

**形式**: `{page_name}_{field_name}_hint` / `{page_name}_{detail}_help`

例:

- `account_new_password_hint` - 8 文字以上で入力してください
- `password_reset_new_email_not_received_help` - メールが届かない場合は迷惑メールフォルダをご確認ください

### リンクテキスト

**形式**: `{page_name}_{purpose}_link` (リンク元ページにスコープして定義する)

例:

- `sign_in_new_forgot_password_link` - パスワードを忘れた？ (sign_in/new からパスワードリセットへ)
- `sign_in_new_sign_up_link` - 新規登録 (sign_in/new からサインアップへ)
- `password_reset_new_back_to_sign_in` - ログインに戻る (password_reset/new からログインへ)

### バリデーションエラー (`validation_*`)

**形式**: `validation_{noun}_{adjective}` または `validation_{condition}`

複数ページから再利用される入力検証エラーは `validation_*` 名前空間に集約する。各ページに展開しないことで、汎用メッセージ (例: 「入力してください」) を 1 箇所で管理できる。

例:

- `validation_required` - 入力してください
- `validation_email_invalid` - 正しいメールアドレスを入力してください
- `validation_password_too_short` - 8 文字以上で入力してください
- `validation_atname_already_taken` - このアットネームは既に使用されています
- `validation_credentials_invalid` - メールアドレスかパスワードが間違っています
- `validation_turnstile_failed` - ロボット検証に失敗しました
- `validation_rate_limit_exceeded` - リクエストが多すぎます

### フラッシュメッセージ (`flash_*`)

**形式**: `flash_{event}`

操作完了後にリダイレクト先で表示する flash メッセージは `flash_*` 名前空間に集約する。命名は「何が起きたか」を表す動詞中心のキーにし、呼び出し元ハンドラー名に縛られない。

例:

- `flash_sign_in_success` - ログインしました
- `flash_sign_out_success` - ログアウトしました
- `flash_account_created` - アカウントを作成しました
- `flash_password_updated` - パスワードを更新しました
- `flash_dismiss` - 閉じる (flash コンポーネント全体で使う UI 部品。これは共通 UI 要素として `flash_*` に含めて運用)

### 共通 UI 要素 (名前空間なし)

サイト全体のデフォルト・エラーページ・メール件名など、特定ページにもどの namespace にも属さないキーは namespace なしで定義する。

例:

- `default_title` - デフォルトのページタイトル
- `default_description` - デフォルトのページ説明
- `error_not_found_title` / `error_not_found_message` / `error_not_found_back_to_home` - 404 エラーページ
- `error_bad_gateway_title` / `error_bad_gateway_message` - 502 エラーページ
- `email_confirmation_subject` - メール確認コードの件名
- `password_reset_email_subject` - パスワードリセットメールの件名

## バリデーションエラーメッセージの国際化

Validator 内で生成するエラーメッセージは必ず `validation_*` 名前空間のキーを使う。

### 例

```go
// ❌ NG: ハードコードされた日本語メッセージ
ve.AddField("email", "メールアドレスを入力してください")

// ❌ NG: ページ固有キーをバリデーションエラーに使う (重複が増える)
ve.AddField("email", i18n.T(ctx, "password_reset_new_email_required"))

// ✅ OK: validation_* 名前空間に集約
ve.AddField("email", i18n.T(ctx, "validation_required"))
```

## ロケールの取得と設定

### ロケールの取得

```go
// 現在のロケールを取得
locale := i18n.GetLocale(ctx)  // "ja" または "en"
```

### ロケールの設定

```go
// ミドルウェアで自動的に設定される
// Accept-Languageヘッダーまたはクッキーから判定
```

### テンプレートでロケールを使用

```templ
templ Default(ctx context.Context, meta viewmodel.PageMeta, content templ.Component) {
    <!DOCTYPE html>
    <html lang={ templates.Locale(ctx) }>
        <head>
            <!-- ... -->
        </head>
        <body>
            @content
        </body>
    </html>
}
```

## テスト

### 翻訳のテスト

```go
func TestTranslations(t *testing.T) {
    tests := []struct {
        name     string
        locale   string
        key      string
        expected string
    }{
        {
            name:     "Japanese password reset title",
            locale:   "ja",
            key:      "password_reset_new_title",
            expected: "パスワードを忘れた？",
        },
        {
            name:     "English password reset title",
            locale:   "en",
            key:      "password_reset_new_title",
            expected: "Forgot your password?",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            ctx = i18n.WithLocale(ctx, tt.locale)

            got := i18n.T(ctx, tt.key)
            if got != tt.expected {
                t.Errorf("i18n.T(%q) = %q, want %q", tt.key, got, tt.expected)
            }
        })
    }
}
```

### プレースホルダー付き翻訳のテスト

```go
func TestTranslationsWithPlaceholders(t *testing.T) {
    ctx := context.Background()
    ctx = i18n.WithLocale(ctx, "ja")

    got := i18n.T(ctx, "watchers_count", map[string]any{"Count": 100})
    expected := "100 人がウォッチ"

    if got != expected {
        t.Errorf("got %q, want %q", got, expected)
    }
}
```

## ベストプラクティス

### 1. 翻訳キーは具体的に

```toml
# ❌ Bad: 汎用的すぎる
[error]
other = "エラーが発生しました"

# ✅ Good: 具体的
[validation_credentials_invalid]
other = "メールアドレスまたはパスワードが正しくありません"
```

### 2. 翻訳キーに日本語を含めない

```toml
# ❌ Bad
[パスワードリセット_タイトル]
other = "パスワードを忘れた？"

# ✅ Good
[password_reset_new_title]
other = "パスワードを忘れた？"
```

### 3. description を必ず記述

```toml
# ❌ Bad: description がない
[password_reset_new_title]
other = "パスワードを忘れた？"

# ✅ Good: description がある
[password_reset_new_title]
description = "パスワードリセットページのタイトル"
other = "パスワードを忘れた？"
```

### 4. 複数形の扱い

```toml
# 日本語は複数形がないため、プレースホルダーで対応
[watchers_count]
description = "ウォッチ数の表示"
other = "{{.Count}} 人がウォッチ"

# 英語は単数形・複数形を区別する場合がある
[watchers_count]
description = "Display watchers count"
one = "{{.Count}} watcher"
other = "{{.Count}} watchers"
```

## トラブルシューティング

### 翻訳が表示されない

1. **翻訳キーが間違っている**: ja.toml/en.tomlのキーを確認
2. **ロケールが設定されていない**: `i18n.WithLocale(ctx, "ja")` を確認
3. **TOMLファイルの構文エラー**: TOMLファイルをバリデート

### 翻訳が英語になってしまう

1. **日本語翻訳が未定義**: ja.tomlに翻訳を追加
2. **デフォルトロケールが英語**: ミドルウェアの設定を確認
3. **Accept-Languageヘッダー**: ブラウザの言語設定を確認

### プレースホルダーが展開されない

1. **構文エラー**: `{{.Key}}` の形式を確認
2. **データが渡されていない**: `map[string]any` の内容を確認
3. **キー名の大文字小文字**: 大文字で始まることを確認 (例: `{{.Count}}`)

## 採用しなかった方針

### A. すべての翻訳キーをページ別命名に統一する

「ページごとに完結する」原則を徹底するため、共通メッセージも各ページ別に複製する方針 (例: `sign_in_new_email_required`, `sign_up_new_email_required`, `password_reset_new_email_required` ... のように同一文言を複数キーで定義)。

**不採用の理由**:

- 「入力してください」のような汎用バリデーションメッセージが N ページ分重複し、TOML が冗長になる
- 同じメッセージを複数キーで管理することで翻訳更新時の漏れリスクが生じる
- 「判断コストをゼロに」の原則は守れても「シンプルさ」とトレードオフになる

**代替として採用した方針**: ページ固有テキストは `{page_name}_*`、複数ページから参照される共通メッセージは `validation_*` / `flash_*` 名前空間、サイト全体の共通 UI 要素は名前空間なしで管理するハイブリッド方式とする。

### B. handler 名のみで page*name を構成する (`{handler}*\*`)

ハンドラーが現状フォーム 1 種類しか持たない場合、`{handler}_{action}_*` ではなく `{handler}_*` で十分という判断。例: `sign_in_title` (sign_in/new のフォームタイトル)。

**不採用の理由**:

- ハンドラーに将来別のフォームアクションが追加された際、既存キーをリネームする必要が生じる
- 同一ハンドラー内で複数アクションを持つ場合 (例: `password/edit` と `password/update` のフォーム別ページ) と、単一アクションの場合で命名が分岐し、判断コストが発生する

**代替として採用した方針**: `{handler}_{action}_*` で常に統一する。`{handler}_*` よりキー名が長くなるが、画面とキーが 1:1 で対応し、ハンドラー拡張に伴うリネームを避けられる。
