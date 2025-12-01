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

- Dev Container を使って開発します
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
docker compose logs -f rails-app
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

**重要**: コードをコミットする前に、以下のコマンドを実行して CI が通ることを確認してください：

```sh
# 1. Zeitwerk（オートロード）チェック
bin/rails zeitwerk:check

# 2. Sorbet型定義の更新と型チェック
bin/rails sorbet:update
bin/srb tc

# 3. Rubyコードのリント・フォーマット
bin/standardrb --fix

# 4. ERBリント
bin/erb_lint --lint-all

# 5. Prettier（JavaScript/CSS）
yarn prettier . --check

# 6. ESLint
yarn eslint .

# 7. テストを実行
bin/rspec
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
- **型注釈**: Sorbet の型注釈を可能な限り追加

  ```ruby
  # typed: true
  extend T::Sig

  sig { params(user_id: String).returns(UserRecord) }
  def find_user(user_id)
    UserRecord.find(user_id)
  end
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
- **対象メッセージ**:
  - ページタイトル、見出し、ラベル、ボタンテキスト
  - エラーメッセージ、成功メッセージ
  - ヘルプテキスト、説明文
  - ログメッセージや開発者向けエラーは日本語のままで OK

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

## セキュリティガイドライン

Web アプリケーションのセキュリティは**最優先事項**です。

### 基本対策

- **CSRF 対策**: `protect_from_forgery` がデフォルトで有効、`form_with` ヘルパーを使用
- **XSS 対策**: ERB の自動エスケープを活用、`raw`/`html_safe` は慎重に使用
- **SQL インジェクション対策**: ActiveRecord のプリペアドステートメント、プレースホルダーを使用
- **認証**: bcrypt（`has_secure_password`）で管理
- **Strong Parameters**: すべてのコントローラーで使用

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

## 関連ドキュメント

- **プロジェクト全体のガイド**: [/CLAUDE.md](../CLAUDE.md) - モノレポ構造、共通インフラ、Rails から Go への移行について
- **Go 版のガイド**: [/go/CLAUDE.md](../go/CLAUDE.md) - Go 版の技術スタック、開発環境、コーディング規約
