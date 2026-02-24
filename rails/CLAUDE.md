# Mewst 開発ガイド (Rails 版)

このファイルは、Rails 版 Mewst の開発に関するガイダンスを提供します。

> **Note**: プロジェクト全体の概要、モノレポ構造、共通インフラ（PostgreSQL）については、[/CLAUDE.md](../CLAUDE.md) を参照してください。

## Rails 版の開発方針

Rails 版は既存の本番システムであり、以下の方針で開発・保守を進めています：

- **安定性優先**: 既存機能の動作を維持しながら慎重に改善
- **段階的な Go 版移行**: 機能ごとに段階的に Go 版へ移行
- **共通インフラの継続利用**: PostgreSQL などは Go 版と共有

### Go 版への移行について

現在、Rails 版の機能を段階的に Go 版へ移行中です。Go 版の実装時には、Rails 版のコードを参考にして既存の仕様を理解できます。

## 技術スタック

### バックエンド

- Ruby 3.3.6
- Rails 7.1.x
- PostgreSQL 16.2
- **認証**: bcrypt（`has_secure_password`）
- **OAuth**: Doorkeeper
- **バックグラウンドジョブ**: Good Job
- **シリアライザ**: Alba
- **型チェック**: Sorbet
- **リント**: Standard (RuboCop)
- **テスト**: RSpec

### フロントエンド

- **JavaScript フレームワーク**: Hotwire (Stimulus, Turbo)
- **CSS フレームワーク**: Tailwind CSS + DaisyUI
- **バンドラー**: esbuild
- **パッケージマネージャー**: Yarn
- **リント**: ESLint
- **フォーマッター**: Prettier
- **テンプレート**: ERB
- **コンポーネント**: ViewComponent

### その他

- **国際化**: Rails I18n (日本語・英語)
- **メール送信**: Resend
- **エラー追跡**: Sentry
- **ページネーション**: activerecord_cursor_paginate

## プロジェクト構造

Rails 標準の MVC アーキテクチャに加え、ユースケース層やコンポーネントを導入した構造：

```
/workspace/rails/
├── app/
│   ├── controllers/      # コントローラー（HTTPリクエスト処理）
│   ├── records/          # ActiveRecordモデル
│   ├── models/           # ドメインモデル・バリューオブジェクト
│   ├── use_cases/        # ユースケース（ビジネスロジック）
│   ├── views/            # ビューテンプレート（ERB）
│   ├── components/       # ViewComponent（再利用可能なUIコンポーネント）
│   ├── forms/            # フォームオブジェクト
│   ├── resources/        # APIリソース
│   ├── serializers/      # シリアライザ（Alba）
│   ├── validators/       # カスタムバリデータ
│   ├── helpers/          # ビューヘルパー
│   ├── javascript/       # JavaScriptファイル
│   ├── assets/           # CSS、画像などのアセット
│   ├── jobs/             # バックグラウンドジョブ
│   └── mailers/          # メーラー
├── config/
│   ├── routes.rb         # ルーティング定義
│   ├── application.rb    # アプリケーション設定
│   ├── database.yml      # データベース設定
│   ├── mewst.yml         # Mewst固有の設定
│   ├── initializers/     # 初期化処理
│   └── locales/          # 国際化ファイル
├── db/
│   ├── migrate/          # マイグレーションファイル
│   └── structure.sql     # DBスキーマ（PostgreSQL形式）
├── spec/                 # RSpecテスト
├── sorbet/               # Sorbet型定義
├── openapi/              # OpenAPI定義
├── public/               # 静的ファイル
├── bin/                  # 実行可能スクリプト
├── Gemfile               # Ruby依存関係
├── package.json          # Node.js依存関係
└── Rakefile              # Rakeタスク
```

## 開発環境のセットアップ

> **Note**: 開発環境の基本的なセットアップ手順は [/CLAUDE.md](../CLAUDE.md#開発環境のセットアップ) を参照してください。

- Docker Compose を使って開発します
- Claude Code はコンテナ内で実行されているため、ホスト側のコマンドの実行は不要です
- 共通インフラ（PostgreSQL）は `/docker-compose.yml` で管理されており、ホスト側で起動済みのはずです

### 環境変数の設定

環境変数は`.env.{environment}`ファイルで管理します：

- `.env.development` - 開発環境用
- `.env.test` - テスト環境用
- `.env.local` - ローカル固有の設定（機密情報など、Git管理外）

### ホスト側で実行するコマンド (Claude Code による実行は不要)

```sh
# コンテナ起動
docker compose up

# 特定のサービスのログを確認
docker compose logs -f app
```

### コンテナ内で実行するコマンド (Claude Code が実行できるコマンド)

```sh
# 依存関係のインストール
bundle install
yarn install

# データベースのセットアップ
bin/rails db:create
bin/rails db:schema:load

# 開発サーバー起動
bin/dev

# Railsサーバーのみ起動
make server

# コンソール起動
make console

# テスト実行
make test
# 特定のテストを実行
make test-file FILE=spec/models/work_spec.rb
# E2Eテストを実行（Playwright）
make test-file FILE=spec/system/

# コードフォーマット
make fmt                           # Ruby（自動修正）
yarn prettier --write "**/*.js"    # JavaScript

# リント
make lint                          # Ruby
yarn eslint "**/*.js"              # JavaScript

# Sorbet型チェック
make sorbet

# Zeitwerk（オートロード）チェック
make zeitwerk

# ERBリント
bin/erb_lint --lint-all

# PostgreSQLに接続
psql $DATABASE_URL

# データベースマイグレーション
make db-migrate
make db-rollback    # 最後のマイグレーションをロールバック

# データベースのセットアップ
make db-setup       # DBの作成、スキーマ読み込み、シード実行

# フロントエンドアセットのビルド
yarn build       # JavaScript（本番用、minify有効）
yarn build:css   # CSS（本番用）

# GraphQL APIスキーマのダンプ
make graphql-dump
```

### コミット前に実行するコマンド

**重要**: コードをコミットする前に、以下のコマンドを実行して CI が通ることを確認してください。

**注意**: 環境変数は 1Password CLI 経由で自動設定されるため、`make` コマンドを使用してください。直接 `bin/rails` 等を実行すると環境変数が不足してエラーになります。

```sh
# 1. Zeitwerk（オートロード）チェック
make zeitwerk

# 2. Sorbet型定義の更新と型チェック
make sorbet-update
make sorbet

# 3. Rubyコードのリント・フォーマット
make fmt

# 4. ERBリント
bin/erb_lint --lint-all

# 5. Prettier（JavaScript/CSS）
yarn prettier . --check

# 6. ESLint
yarn eslint .

# 7. テストを実行
make test

# 全ての検証を実行（ワンライナー）
bin/check
```

## Pull Request のガイドライン

Pull Request のガイドラインは [/CLAUDE.md](../CLAUDE.md#pull-requestのガイドライン) を参照してください。

**要約**:

- 実装コード: 300 行以下を目安
- テストコード: 制限なし（必要な分だけ書く）
- 実装とテストは同じ PR に含める
- 「行数を守ること」よりも「きちんと実装すること」を優先

## コーディング規約

### Ruby コード

- **インデント**: 2 スペースを使用（Ruby 標準）
- **スタイルガイド**: Standard（RuboCop）に従う
- **自動フォーマット**: `make fmt`を使用
- **コメント**: 日本語で記述（複雑なロジックの説明）
- **1行100文字以内**: 長い行は適切に改行する
- **型注釈**: Sorbet の型注釈を可能な限り追加

```ruby
# typed: strict
# frozen_string_literal: true

class Example
  # ✅ 文字列はダブルクオート
  name = "example"

  # ✅ ハッシュの省略記法
  {user:, name:}

  # ❌
  {user: user, name: name}

  # ✅ プライベートメソッドは private def
  private def process_value(value)
    value.upcase
  end

  # ✅ プロテクテッドメソッドは protected def
  protected def shared_method(value)
    value.downcase
  end

  # ❌ 後置ifは使用しない
  # return if value.nil? # 悪い例

  # ✅
  if value.nil?
    return
  end

  # ❌ attr_readerにprivateブロックを使用しない
  # private
  # attr_reader :user_record

  # ✅ attr_readerは個別にprivate指定
  attr_reader :user_record
  private :user_record

  # ✅ T.mustではなくnot_nil!を使用
  value.not_nil!
end
```

### ActiveRecord

```ruby
# ❌ includesは使用禁止
Model.includes(:association)

# ✅ 明示的にpreloadまたはeager_loadを使用
Model.preload(:association)   # 別クエリで取得（基本はこちら）
Model.eager_load(:association) # JOINで取得（関連テーブルでフィルタリング時）
```

#### コメントのガイドライン

**良いコメント**：

- コードの**意図や理由**を説明する（「なぜこうしたか」）
- 将来の開発者が理解できる、文脈に依存しない内容
- 複雑なロジックや、一見不自然に見える実装の背景を説明する

```ruby
# 良い例: 意図を説明
# ユーザーが削除済みでも、過去の記録との整合性を保つためにIDは保持する
return user.id if user.discarded?

# 良い例: 制約や前提条件を説明
# NOTE: ULIDを使用しているため、created_atでソートする代わりにidでソートできる
Post.order(id: :desc)
```

**避けるべきコメント**：

- **実装の変遷を説明するコメント**（「以前は〜だった」「〜は削除した」など）
- **過去との比較**（「bundle install に統合したため不要」など）
- **自明なことの説明**（コードを読めばわかること）
- **やり取りの文脈に依存するコメント**（PR レビューのコメントは PR に書く）

```ruby
# 悪い例: 実装の変遷を説明（git履歴で確認できる）
# 以前はここでGemをインストールしていたが、Gemfileに統合したため削除した

# 悪い例: 自明なことを説明
# ユーザーIDを取得
user_id = user.id

# 良い例: 複雑なロジックの意図を説明
# ユーザーIDを取得（削除済みユーザーはnilを返す）
user_id = user.discarded? ? nil : user.id
```

**原則**：

- **コメントはコードの「なぜ」を説明し、「何を」はコードに語らせる**
- git の履歴に残る情報（過去の実装、変更の経緯）はコメントに書かない
- レビューコメントや議論の文脈に依存する内容は書かない

詳細については、[/CLAUDE.md](../CLAUDE.md#コメントのガイドライン) を参照してください。

### テンプレート（ERB）

- **インデント**: 2 スペースを使用
- **リント**: `bin/erb_lint --lint-all` でチェック

### JavaScript

- **インデント**: 2 スペースを使用
- **スタイルガイド**: ESLint に従う
- **フォーマッター**: Prettier
- **フレームワーク**: Stimulus Controller を優先的に使用

### アーキテクチャパターン

Rails 版 Mewst は、標準の MVC アーキテクチャに加え、以下のパターンを導入しています：

#### Records（ActiveRecord モデル）

データベーステーブルに対応する ActiveRecord モデルを配置します。

- **配置**: `app/records/`
- **命名**: `{Model}Record`（例: `UserRecord`, `PostRecord`）
- **責務**: データの永続化、リレーション定義、基本的なバリデーション

#### Use Cases（ユースケース）

ビジネスロジックを担当します。

- **配置**: `app/use_cases/`
- **命名**: `{Action}{Entity}UseCase`（例: `CreatePostUseCase`, `FollowProfileUseCase`）
- **メソッド**: `call` メソッドを実装

#### ViewComponent

再利用可能な UI コンポーネントを実装します。

- **配置**: `app/components/`
- **命名**: `{ComponentName}Component`
- **テンプレート**: ERB を使用

### RSpec コーディング規約

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

#### システムテストの待機処理

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

### クラス間の依存関係ルール

| クラス     | 依存可能な先                                      |
| ---------- | ------------------------------------------------- |
| Component  | Component, Form, Model                            |
| Controller | Form, Model, Record, Repository, UseCase, View    |
| Form       | Record, Validator                                 |
| Job        | UseCase                                           |
| Mailer     | Model, Record, Repository, View                   |
| Model      | Model                                             |
| Policy     | Record                                            |
| Record     | Record                                            |
| Repository | Model, Record, Policy                             |
| UseCase    | Job, Mailer, Record                               |
| Validator  | Record                                            |
| View       | Component, Form, Model                            |

#### UseCaseとJobの依存関係について

UseCaseとJobの間には相互依存が存在しますが、以下のルールで循環依存を回避します：

- **UseCase → Job**: `perform_later`メソッドによるキューへの追加のみ許可
- **Job → UseCase**: ジョブ実行時のUseCase呼び出しは許可
- **重要**: UseCaseからJobインスタンスの直接実行（`perform`メソッド）は禁止

```ruby
# ✅ 良い例：UseCaseからジョブをキューに追加
class Users::CreateUseCase < ApplicationUseCase
  def call
    user = UserRecord.create!(...)
    Users::SendWelcomeEmailJob.perform_later(user.id)  # キューに追加のみ
  end
end

# ❌ 悪い例：UseCaseからジョブを直接実行
class Users::CreateUseCase < ApplicationUseCase
  def call
    user = UserRecord.create!(...)
    Users::SendWelcomeEmailJob.new.perform(user.id)  # 直接実行は禁止
  end
end
```

#### UseCaseクラスを使用する場合

- ✅ データベースへの永続化を伴う処理
- ✅ 複数のモデル/レコードにまたがる複雑なビジネスロジックで永続化を伴うもの
- ✅ トランザクション管理が必要な処理

#### UseCaseクラスを使用しない場合

- ❌ データベースへの永続化を伴わない処理（URL生成、データ変換など）
- ❌ 単一のモデル/レコードに閉じた処理（モデルやレコードのメソッドとして定義）

#### トランザクション処理

**重要**: UseCaseクラスでトランザクションを張る場合は、必ず `#with_transaction` メソッドを使用すること

```ruby
# ✅ 良い例：with_transactionを使用
class Users::CreateUseCase < ApplicationUseCase
  def call
    with_transaction do
      user = UserRecord.create!(...)
      ProfileRecord.create!(user:, ...)
    end
  end
end

# ❌ 悪い例：transactionを直接使用
class Users::CreateUseCase < ApplicationUseCase
  def call
    ApplicationRecord.transaction do
      # with_transactionを使うべき
    end
  end
end
```

**重要**: Controller、Job、Rakeタスク内で永続化処理を書く場合は、必ずUseCaseクラスを定義すること

### 命名規則

- Controller: `(ModelPlural)::(ActionName)Controller`
- UseCase: `(ModelPlural)::(Verb)UseCase`
- Form: `(ModelPlural)::(Noun)Form`
- Repository: `(Model)Repository`
- View: `(ModelPlural)::(ActionName)View`
- Component: `(UIComponentPlural)::(Noun)Component`

### 国際化（I18n）

すべてのユーザー向けメッセージは**必ず国際化対応**します：

- **対応言語**: 日本語と英語
- **翻訳ファイル**: `config/locales/` 配下に配置
  - `messages.ja.yml`, `messages.en.yml` - メッセージ
  - `nouns.ja.yml`, `nouns.en.yml` - 名詞
  - `verbs.ja.yml`, `verbs.en.yml` - 動詞
  - `forms.ja.yml`, `forms.en.yml` - フォーム
  - `meta.ja.yml`, `meta.en.yml` - メタ情報
- **ビュー**: `t('.message_key')` または `I18n.t('message_key')` で翻訳を呼び出す
- **モデル**: `human_attribute_name` でカラム名を国際化
- **対象メッセージ**:
  - ページタイトル、見出し、ラベル、ボタンテキスト
  - エラーメッセージ、成功メッセージ
  - ヘルプテキスト、説明文
  - ログメッセージや開発者向けエラーは日本語のままで OK

#### バリデーションエラーメッセージの国際化

ActiveRecord のバリデーションエラーメッセージも国際化します：

```yaml
# config/locales/ja.yml
ja:
  activerecord:
    errors:
      models:
        user_record:
          attributes:
            email:
              blank: "を入力してください"
              invalid: "の形式が正しくありません"
```

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

## 重要な原則

- ネストしたトランザクションを避ける
- レコードのコールバックを避ける
- View/Componentでのデータベースアクセスを防ぐ
- 問題が解決されるなら、レイヤーを跨いだ依存も許可
- 説明的な命名規則
- コメントは日本語で記載
- 1行100文字以内

## セキュリティガイドライン

Web アプリケーションのセキュリティは**最優先事項**です。

### 基本対策

- **CSRF 対策**: `protect_from_forgery` がデフォルトで有効、`form_with` ヘルパーを使用
- **XSS 対策**: ERB の自動エスケープを活用、`raw`/`html_safe` は慎重に使用
- **SQL インジェクション対策**: ActiveRecord のプリペアドステートメント、プレースホルダーを使用
- **認証**: bcrypt（`has_secure_password`）で管理
- **Strong Parameters**: すべてのコントローラーで使用

## 作業完了ガイドライン

### タスク実装フロー

#### 1. タスク理解

- 要件を理解
- このガイドの固有ルールを確認

#### 2. 実装前の準備

- 既存コードの調査
- 特に以下を意識：
  - Records には `{Model}Record` の命名規則
  - Use Cases には `call` メソッドを実装

#### 3. 実装

- 規約に従ってコーディング

#### 4. 完了前の検証

**重要**: 完了報告前に全ての作業が適切に検証されていることを確認すること

- **テスト作成**: テスト作成後は、必ず `make test` を実行してテストが通ることを確認する
- **コード実装**: コード記述後は、必ず以下を確認する:
  - 型チェックが成功すること（`make sorbet`）
  - Linter の実行が成功すること（`make fmt`, `bin/erb_lint --lint-all`）
  - 関連するテストが通ること（`make test-file FILE=spec/path/to/xxx_spec.rb`）
  - 明らかなランタイムエラーがないこと
- **ドキュメント編集**: Markdown ファイルを編集した後は、必ず以下を行う:
  - Prettier の実行 (`yarn prettier . --write`)
- **リトライポリシー**: 問題発生時は自動で最大 5 回まで再試行し、それでも解消できない場合にのみユーザーへ連絡する（途中経過は報告しない）
- **以下の状態では絶対に完了報告をしない**:
  - テストが失敗している（未実装機能のテストを意図的に作成している場合を除く）
  - コンパイルエラーがある
  - 前回の試行から未解決のエラーが残っている

### 検証コマンド

```bash
# Ruby のファイルを編集したとき実行する
make fmt
bin/erb_lint --lint-all
make sorbet
make sorbet-update
make zeitwerk
make test

# JavaScript を編集したとき実行する
yarn prettier . --write
yarn eslint .

# 全ての検証を実行
bin/check
```

## データベース管理

### マイグレーション

```sh
# 新しいマイグレーションを作成
bin/rails generate migration CreatePosts

# マイグレーションを実行
make db-migrate

# マイグレーションをロールバック
make db-rollback

# スキーマをダンプ（structure.sql）
make db-migrate
```

### スキーマ管理

- **スキーマファイル**: `db/structure.sql` （PostgreSQL 形式）
- **Go 版との共有**: Rails 版のマイグレーションが DB スキーマを管理
- **ULID**: 主キーには ULID を使用（`generate_ulid()` 関数で生成）

## デバッグ・トラブルシューティング

よくある問題とその解決方法：

- **Sorbet エラー**: `make sorbet-update` で型定義を更新
- **オートローディングエラー**: `make zeitwerk` でチェック
- **フォーマットエラー**: `make fmt` または `yarn prettier . --write` で修正
- **Lint エラー**: 各種 Lint コマンド（`make lint`, `bin/erb_lint --lint-all`, `yarn eslint .`）で修正

## 関連ドキュメント

- **プロジェクト全体のガイド**: [/CLAUDE.md](../CLAUDE.md) - モノレポ構造、共通インフラ、Rails から Go への移行について
- **Go 版のガイド**: [/go/CLAUDE.md](../go/CLAUDE.md) - Go 版の技術スタック、開発環境、コーディング規約
