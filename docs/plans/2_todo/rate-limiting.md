# レート制限の追加 設計書

## 実装ガイドラインの参照

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド
- [@go/docs/handler-guide.md](/workspace/go/docs/handler-guide.md) - HTTPハンドラーガイドライン
- [@go/docs/security-guide.md](/workspace/go/docs/security-guide.md) - セキュリティガイドライン

## 概要

Go 版 Mewst の既存エンドポイントに PostgreSQL ベースのレート制限を追加します。ブルートフォース攻撃やメール送信スパムを防止し、サービスの安定性とセキュリティを向上させます。

**目的**:

- ブルートフォース攻撃（ログイン、確認コード検証）を防止する
- メール送信のスパム（パスワードリセット）を防止する
- サービスの安定性を確保する

**背景**:

- 現在、Go 版のエンドポイントには Turnstile による Bot 対策のみ実施
- Turnstile を突破された場合の防御層としてレート制限が必要
- Wikino で実装した PostgreSQL ベースのレート制限を Mewst にも適用する

**前提条件**:

- サインアップ機能の実装（タスク 1-3）でレート制限の基盤（`rate_limits` テーブル、`ratelimit` パッケージ）が作成済みであること

## 要件

### 機能要件

- ログイン（`POST /sign_in`）に IP アドレスベースのレート制限を適用する
- パスワードリセット（`POST /password_reset`）に IP アドレス + メールアドレスベースのレート制限を適用する
- メール確認（`POST /email_confirmation`）に IP アドレスベースのレート制限を適用する
- レート制限超過時は HTTP 429 Too Many Requests を返す
- レート制限超過時はユーザーにわかりやすいエラーメッセージを表示する

### 非機能要件

#### セキュリティ

- レート制限の設定値は環境変数で変更可能にする（デフォルト値はハードコード）
- レート制限超過時のログ出力（不正アクセスの検知に利用）

#### 国際化

- エラーメッセージは日本語・英語の両言語に対応

## 設計

### レート制限の設定

| エンドポイント             | キー           | 制限  | ウィンドウ |
| -------------------------- | -------------- | ----- | ---------- |
| `POST /sign_in`            | IP アドレス    | 10 回 | 15 分      |
| `POST /password_reset`     | IP アドレス    | 5 回  | 15 分      |
| `POST /password_reset`     | メールアドレス | 3 回  | 15 分      |
| `POST /email_confirmation` | IP アドレス    | 10 回 | 15 分      |

### コード設計

#### ハンドラーへの統合

各ハンドラーの `Create` メソッド内でレート制限をチェックします。

```go
// sign_in/create.go の例
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // レート制限チェック（Turnstile検証の前に実施）
    ip := middleware.GetClientIP(r)
    result, err := h.rateLimiter.Check(ctx, ratelimit.CheckInput{
        Key:    ratelimit.IPKey(ip),
        Limit:  10,
        Window: 15 * time.Minute,
    })
    if err != nil {
        // エラーログ出力、500エラー
        return
    }
    if !result.Allowed {
        // 429 Too Many Requests
        h.renderRateLimitError(w, r)
        return
    }

    // 以降の処理...
}
```

#### Handler 構造体への依存追加

```go
type Handler struct {
    // 既存のフィールド...
    rateLimiter *ratelimit.Limiter
}
```

### エラーレスポンス

レート制限超過時は、フォームを再表示しエラーメッセージを表示します。

```go
func (h *Handler) renderRateLimitError(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusTooManyRequests)
    // フォームを再表示し、エラーメッセージを表示
}
```

### 国際化

```toml
# ja.toml
rate_limit_exceeded = "リクエストが多すぎます。しばらく時間をおいてから再度お試しください。"

# en.toml
rate_limit_exceeded = "Too many requests. Please wait a moment and try again."
```

## タスクリスト

### フェーズ 1: ログインへのレート制限追加

- [ ] **1-1**: [Go] ログインハンドラーへのレート制限追加
  - `internal/handler/sign_in/handler.go` に `rateLimiter` を追加
  - `internal/handler/sign_in/create.go` でレート制限チェックを実装
  - 国際化メッセージの追加（`ja.toml`, `en.toml`）
  - テストの追加
  - **想定ファイル数**: 約 5 ファイル（実装 4 + テスト 1）
  - **想定行数**: 約 150 行（実装 80 行 + テスト 70 行）

### フェーズ 2: パスワードリセットへのレート制限追加

- [ ] **2-1**: [Go] パスワードリセットハンドラーへのレート制限追加
  - `internal/handler/password_reset/handler.go` に `rateLimiter` を追加
  - `internal/handler/password_reset/create.go` でレート制限チェックを実装（IP + メールアドレスの二重チェック）
  - テストの追加
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 120 行（実装 60 行 + テスト 60 行）

### フェーズ 3: メール確認へのレート制限追加

- [ ] **3-1**: [Go] メール確認ハンドラーへのレート制限追加
  - `internal/handler/email_confirmation/handler.go` に `rateLimiter` を追加
  - `internal/handler/email_confirmation/create.go` でレート制限チェックを実装
  - テストの追加
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 100 行（実装 50 行 + テスト 50 行）

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **グローバルレート制限（全エンドポイント共通）**: 必要性が明確になった時点で検討
- **ユーザーIDベースのレート制限**: 認証済みユーザー向けの制限は別途検討
- **レート制限の動的設定変更**: 環境変数での設定変更で十分
- **レート制限ダッシュボード**: 管理画面は別タスクで実装予定

## 参考資料

- **Wikino レート制限実装**: `/wikino/go/internal/ratelimit/limiter.go`
- **Mewst サインアップ設計書**: `/workspace/docs/designs/1_doing/sign-up.md`
- **Go 版 Mewst ログイン実装**: `/workspace/go/internal/handler/sign_in/`
- **Go 版 Mewst パスワードリセット実装**: `/workspace/go/internal/handler/password_reset/`
- **Go 版 Mewst メール確認実装**: `/workspace/go/internal/handler/email_confirmation/`
