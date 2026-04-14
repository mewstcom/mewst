# セキュリティガイドライン

このドキュメントは、Rails 版 Mewst でのセキュリティベストプラクティスを説明します。

## 基本方針

Web アプリケーションのセキュリティは**最優先事項**です。以下のガイドラインを必ず守ってください。

## CSRF（Cross-Site Request Forgery）対策

### Rails 標準の保護

Rails では `protect_from_forgery` がデフォルトで有効になっています。

```ruby
# app/controllers/application_controller.rb
class ApplicationController < ActionController::Base
  # protect_from_forgery はデフォルトで有効
end
```

### フォームでの使用

`form_with` ヘルパーが自動的に CSRF トークンを追加します。

```ruby
# ✅ Good: form_withは自動的にCSRFトークンを追加
<%= form_with url: sessions_path do |f| %>
  <%= f.text_field :email %>
  <%= f.submit "ログイン" %>
<% end %>

# ❌ Bad: 手動でformタグを書く（CSRFトークンが含まれない）
<form action="/sessions" method="post">
  <input type="text" name="email" />
</form>
```

## XSS（Cross-Site Scripting）対策

### テンプレートの自動エスケープ

ERB は自動でエスケープ処理を行います。

```ruby
# ✅ 自動的にエスケープされる
<%= @user.comment %>  # <script>...</script> は &lt;script&gt; になる
```

### raw/html_safe の注意

`raw` や `html_safe` を使う場合は、データが信頼できるソースからのものであることを確認してください。

```ruby
# ⚠️ 注意: 信頼できるHTMLのみ
<%= raw @trusted_html_content %>

# ❌ NG: ユーザー入力を直接raw()で使用しない
<%= raw @user.comment %>

# ✅ Good: サニタイズを使用
<%= sanitize @user.comment, tags: %w[p br strong em] %>
```

### Content Security Policy

`config/initializers/content_security_policy.rb` で CSP を設定できます。現在はコメントアウトされていますが、必要に応じて有効化してください。

```ruby
# config/initializers/content_security_policy.rb
Rails.application.configure do
  config.content_security_policy do |policy|
    policy.default_src :self, :https
    policy.font_src    :self, :https, :data
    policy.img_src     :self, :https, :data
    policy.object_src  :none
    policy.script_src  :self, :https
    policy.style_src   :self, :https
  end
end
```

## SQL インジェクション対策

### ActiveRecord のプリペアドステートメント

ActiveRecord は自動的にプリペアドステートメントを使用します。

```ruby
# ✅ Good: 安全
UserRecord.where(email: params[:email])
UserRecord.where("email = ?", params[:email])
UserRecord.where("email = :email", email: params[:email])

# ❌ NG: SQLインジェクションの脆弱性
UserRecord.where("email = '#{params[:email]}'")
```

### where 条件のプレースホルダー

必ずプレースホルダー（`?` または名前付き）を使用します。

```ruby
# ✅ Good: プレースホルダーを使用
private def search_posts(keyword)
  PostRecord.where("body LIKE ?", "%#{sanitize_sql_like(keyword)}%")
end

# ❌ NG: 文字列補間
private def search_posts(keyword)
  PostRecord.where("body LIKE '%#{keyword}%'")
end
```

## パスワード管理

### has_secure_password の使用

Rails 版では `has_secure_password` を使用してパスワードを管理します。

```ruby
# app/records/user_record.rb
class UserRecord < ApplicationRecord
  PASSWORD_MIN_LENGTH = 8

  has_secure_password

  validates :email, presence: true, uniqueness: true
end

# パスワードの検証
user.authenticate("password")  # 成功時: user, 失敗時: false
```

### 平文パスワードの扱い

```ruby
# ❌ NG: 平文パスワードをログに出力
Rails.logger.info "User password: #{params[:password]}"

# ✅ Good: パスワードはログに出力しない
Rails.logger.info "User login attempt: #{params[:email]}"
```

### パラメータフィルタリング

`config/initializers/filter_parameter_logging.rb` でパスワード等の機密情報がログに出力されないよう設定されています。

```ruby
# config/initializers/filter_parameter_logging.rb
Rails.application.config.filter_parameters += [
  :passw, :secret, :token, :_key, :crypt, :salt, :certificate, :otp, :ssn
]
```

## 認証・認可

### 認証

`ControllerConcerns::Authenticatable` モジュールでセッションベースの認証を提供しています。

```ruby
# 認証が必要なコントローラーで before_action を設定
class Home::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable

  before_action :require_authentication

  def call
    # viewer でログイン中のユーザー（ActorRecord）を取得
    @actor = viewer!
  end
end
```

**主なメソッド**:

| メソッド                   | 用途                                            |
| -------------------------- | ----------------------------------------------- |
| `viewer`                   | ログイン中のユーザーを取得（未ログイン時は nil） |
| `viewer!`                  | ログイン中のユーザーを取得（未ログイン時はエラー）|
| `signed_in?`               | ログイン済みかどうかを判定                       |
| `require_authentication`   | 未ログイン時にリダイレクト                       |
| `require_no_authentication`| ログイン済み時にリダイレクト                     |

### 認可（リソースの所有者チェック）

ログインしているだけでは不十分です。ユーザーが操作権限を持つリソースかチェックします。

```ruby
# ✅ Good: リソースの所有者チェック
class Posts::DestroyController < ApplicationController
  include ControllerConcerns::Authenticatable

  before_action :require_authentication

  def call
    post_record = PostRecord.find(params[:id])

    # 所有者チェック
    unless post_record.actor_id == viewer!.id
      head :forbidden
      return
    end

    # 削除処理
    post_record.destroy!
    redirect_to root_path
  end
end
```

## Strong Parameters

すべてのコントローラーで Strong Parameters を使用します。

```ruby
# ✅ Good: Strong Parameters
private def post_params
  params.require(:post).permit(:body)
end

# ❌ NG: パラメータを直接使用
PostRecord.new(params[:post])  # Mass assignment vulnerability
```

## セッション管理

### セッションの構成

Mewst では 2 種類のセッション機構を使用しています：

| 種類                     | 用途                       | ストレージ            |
| ------------------------ | -------------------------- | --------------------- |
| Rails セッション         | フラッシュメッセージ等     | Cookie（`_mewst_session`） |
| 認証セッショントークン   | ユーザー認証               | DB（`sessions` テーブル）+ Cookie |

### 認証セッションの仕組み

1. ログイン成功時に `SessionRecord` を作成し、セキュアトークンを生成
2. トークンを Cookie（`mewst_session_token`）に保存
3. リクエストごとにトークンで `SessionRecord` を検索し、ユーザーを特定

```ruby
# app/records/session_record.rb
class SessionRecord < ApplicationRecord
  COOKIE_KEY = :mewst_session_token

  has_secure_token

  belongs_to :actor_record, class_name: "ActorRecord", foreign_key: :actor_id
end
```

### Cookie のセキュリティ設定

認証 Cookie は以下の属性で設定されています：

```ruby
# app/controllers/controller_concerns/authenticatable.rb
cookies.permanent[SessionRecord::COOKIE_KEY] = {
  value: session.token,
  httponly: true,     # JavaScriptからアクセス不可
  same_site: :lax     # CSRF対策
}
```

- **HttpOnly**: JavaScript からの Cookie アクセスを防止（XSS によるトークン窃取を防ぐ）
- **SameSite: Lax**: クロスサイトリクエストでの Cookie 送信を制限
- **Secure**: 本番環境では `force_ssl` により HTTPS 経由のみ Cookie が送信される

### Go 版とのセッション共有

認証セッションは Go 版と同一の `sessions` テーブルを共有しています。セッション管理の全体像については [/CLAUDE.md](../CLAUDE.md#セッションストアpostgresql) を参照してください。

## エラーメッセージ

### 詳細な情報を漏らさない

システムの内部構造や SQL エラーをユーザーに見せないようにします。

```ruby
# ❌ NG: 詳細なエラーメッセージをユーザーに表示
rescue ActiveRecord::RecordNotFound => e
  render plain: e.message, status: :not_found
end

# ✅ Good: 一般的なエラーメッセージを表示し、詳細はログに記録
rescue ActiveRecord::RecordNotFound
  head :not_found
end

rescue => e
  Rails.logger.error "Error: #{e.message}"
  Rails.logger.error e.backtrace.join("\n")
  head :internal_server_error
end
```

### Sentry でエラー追跡

本番環境では Sentry でエラーを追跡しています。

```ruby
# config/initializers/sentry.rb
Sentry.init do |config|
  config.dsn = Rails.configuration.mewst["sentry_dsn"]
  config.breadcrumbs_logger = %i[active_support_logger http_logger]
  config.traces_sample_rate = 0.5
end
```

## セキュリティヘッダー

### Rails デフォルトのヘッダー

Rails 7.1 ではデフォルトで以下のセキュリティヘッダーが設定されています：

| ヘッダー                            | 値                               | 効果                         |
| ----------------------------------- | -------------------------------- | ---------------------------- |
| `X-Frame-Options`                   | `SAMEORIGIN`                     | クリックジャッキング防止     |
| `X-Content-Type-Options`            | `nosniff`                        | MIME タイプスニッフィング防止|
| `X-Permitted-Cross-Domain-Policies` | `none`                           | Flash/PDF の埋め込み防止     |
| `Referrer-Policy`                   | `strict-origin-when-cross-origin`| リファラー情報の制限         |

### HSTS ヘッダー（本番環境）

```ruby
# config/environments/production.rb
config.force_ssl = true  # HSTSヘッダーを自動設定、HTTPSを強制
```

`force_ssl` により以下が有効になります：

- HTTP リクエストを HTTPS にリダイレクト
- `Strict-Transport-Security` ヘッダーの送信
- Cookie に `Secure` フラグを自動付与

## セキュリティチェックリスト

新機能を実装する際は、以下を必ず確認してください：

### フォーム送信

- [ ] `form_with` ヘルパーを使用しているか
- [ ] CSRF トークンが含まれているか

### ユーザー入力

- [ ] Strong Parameters を使用しているか
- [ ] バリデーションを実施しているか
- [ ] ホワイトリスト方式で許可しているか

### データベース

- [ ] ActiveRecord を使用しているか
- [ ] プレースホルダーを使用しているか
- [ ] 文字列補間を避けているか

### パスワード

- [ ] `has_secure_password` を使用しているか
- [ ] 平文パスワードをログに出力していないか
- [ ] パラメータフィルタリングを設定しているか

### 認証・認可

- [ ] `require_authentication` を設定しているか
- [ ] リソースの所有者チェックを行っているか

### エラー処理

- [ ] エラーメッセージは適切か（詳細な情報を漏らしていないか）
- [ ] Sentry でエラーを追跡しているか

## ベストプラクティス

### 1. Bundler Audit で脆弱性をチェック

```sh
# 定期的に実行
bundle audit check --update
```

### 2. Brakeman で静的解析

```sh
# セキュリティ脆弱性をスキャン
brakeman -q
```

### 3. 環境変数でシークレットを管理

```ruby
# ❌ NG: ハードコード
API_KEY = "abc123"

# ✅ Good: 環境変数
API_KEY = ENV["API_KEY"]
```

### 4. 定期的に Gem を更新

```sh
# 定期的に実行
bundle update
```

## トラブルシューティング

### CSRF トークンエラー

**症状**: "Can't verify CSRF token authenticity"

**原因**:

1. フォームに CSRF トークンが含まれていない
2. セッションが切れている
3. AJAX リクエストにトークンが含まれていない

**解決方法**:

```ruby
# フォーム: form_withを使用
<%= form_with url: posts_path do |f| %>
  <%= f.submit %>
<% end %>

# Turbo/Stimulus: CSRFトークンはmetaタグから自動取得される
# <meta name="csrf-token" content="..." /> がlayoutに含まれていることを確認
```

### Mass Assignment

**症状**: 想定外のパラメータが保存されてしまう

**原因**: Strong Parameters を使用していない

**解決方法**:

```ruby
# ✅ Good: Strong Parameters
private def post_params
  params.require(:post).permit(:body)
end
```

## 参考資料

- [Rails Security Guide](https://guides.rubyonrails.org/security.html)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Brakeman](https://brakemanscanner.org/)
- [Bundler Audit](https://github.com/rubysec/bundler-audit)
