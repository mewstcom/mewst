# テストガイド

このドキュメントは、Rails 版 Mewst のテスト戦略と RSpec コーディング規約を説明します。

## RSpec コーディング規約

```ruby
# ❌ context, let, described_classは使用しない
context "when xxx" do
  let(:user) { create(:user) }
end

# ✅ itブロック内で変数定義
it "xxxのとき、somethingすること" do
  user = FactoryBot.create(:user)
  # テスト実装
end

# ✅ FactoryBotで作成したレコードの変数名には_recordサフィックスを付ける
user_record = FactoryBot.create(:user_record)
post_record = FactoryBot.create(:post_record)

# ❌ サフィックスなしの変数名は避ける
user = FactoryBot.create(:user_record)
```

### システムテストの待機処理

```ruby
# ❌ sleepを使用した待機処理は避ける
button.click
sleep 2
expect(page).to have_current_path(some_path)

# ✅ Capybaraの待機機能を活用
button.click
# ページ上の要素の変化を待つ（Capybaraが自動的に最大5秒待機）
expect(page).not_to have_content("削除されたコンテンツ")
expect(page).to have_content("新しく表示されるコンテンツ")

# ✅ have_css/not_to have_cssで要素の出現/消失を待つ
expect(page).to have_css(".success-message")
expect(page).not_to have_css(".loading-spinner")
```

**重要**: システムテストでは`sleep`の使用を避け、Capybaraの自動待機機能を活用すること

## テスト戦略

Rails 版 Mewst は、RSpec を使用した包括的なテストを実施しています。

### 基本方針

- **テストファースト**: 実装前にテストを書くことを推奨
- **実データベースを使用**: 基本的にデータベースをモックせず、実際の PostgreSQL を使用
- **FactoryBot**: テストデータは FactoryBot で作成

### テストの種類

- **モデルテスト**: `spec/models/` - バリデーション、メソッドの動作確認
- **リクエストテスト**: `spec/requests/` - HTTP リクエスト・レスポンス、認証・認可
- **システムテスト**: `spec/system/` - ブラウザを使った E2E テスト（Capybara + Cuprite）
- **フォームテスト**: `spec/forms/` - フォームオブジェクトのテスト
- **ユースケーステスト**: `spec/use_cases/` - ユースケースのテスト

### テストの実行

```sh
# 全テスト実行
bin/rspec

# 特定のファイルを実行
bin/rspec spec/requests/posts_spec.rb

# 特定の行を実行
bin/rspec spec/requests/posts_spec.rb:10

# システムテストを実行
bin/rspec spec/system/
```
