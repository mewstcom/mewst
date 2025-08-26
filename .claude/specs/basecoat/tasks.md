# Basecoat移行タスクリスト

daisyUIからBasecoatへの移行タスクリスト。各ページのCSSを確認しながら段階的に移行します。

## HTMLビューを持つページ（優先度高）

### 認証・アカウント関連
- [ ] `/sign_in` - サインインページ (sessions/new#call)
- [ ] `/sign_up` - サインアップページ (sign_up/new#call)
- [ ] `/accounts/new` - アカウント新規作成ページ (accounts/new#call)
- [ ] `/password_reset` - パスワードリセットページ (password_resets/new#call)
- [ ] `/password/edit` - パスワード編集ページ (passwords/edit#call)
- [ ] `/email_confirmations/new` - メール確認ページ (email_confirmations/new#call)

### メインページ
- [ ] `/` - トップページ (welcome/show#call)
- [ ] `/home` - ホームページ (home/show#call)
- [ ] `/public` - パブリックタイムライン (public/show#call)
- [ ] `/community` - コミュニティページ (communities/show#call)

### プロフィール・投稿関連
- [ ] `/@:atname` - プロフィールページ (profiles/show#call)
- [ ] `/@:atname/posts/:post_id` - 投稿詳細ページ (posts/show#call)
- [ ] `/new` - 新規投稿ページ (posts/new#call)

### 検索・探索
- [ ] `/search` - 検索ページ (search/show#call)
- [ ] `/search/profiles` - プロフィール検索結果 (search/profiles/index#call)
- [ ] `/followees` - フォロー一覧 (followees/index#call)

### 設定
- [ ] `/settings` - 設定トップページ (settings/index#call)
- [ ] `/settings/profile` - プロフィール設定 (settings/profiles/show#call)
- [ ] `/settings/email` - メール設定 (settings/emails/show#call)
- [ ] `/settings/user` - ユーザー設定 (settings/users/show#call)

### その他
- [ ] `/notifications` - 通知一覧 (notifications/index#call)
- [ ] `/links/new` - リンク追加ページ (links/new#call)
- [ ] `/privacy` - プライバシーポリシー (privacies/show#call)
- [ ] `/terms` - 利用規約 (terms/show#call)

## API・フィード（優先度低）
- [ ] `/@:atname/atom` - Atomフィード (profiles/atom/show#call)
- [ ] `/manifest` - マニフェスト (manifests/show#call)

## APIエンドポイント（フロントエンド変更不要）
以下のAPIエンドポイントはJSONレスポンスのため、CSS変更の対象外：
- `/api/v1/@:atname/posts`
- `/api/v1/internal/@:atname/posts`
- `/api/v1/internal/email_confirmations/:email_confirmation_id`
- `/api/v1/internal/posts/:post_id`
- `/api/v1/notifications`
- `/api/v1/posts/:post_id`
- `/api/v1/profiles/me`
- `/api/v1/suggested_profiles`
- `/api/v1/timeline`
- `/api/v1/users/me`

## 進捗管理
- 完了したタスクには `[x]` をマークする
- 各ページの移行時に影響を受けるコンポーネントも同時に更新する
- 共通レイアウトやコンポーネントの変更は慎重に行う