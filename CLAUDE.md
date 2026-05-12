# Mewst 開発ガイド

このファイルは、Claude Code (claude.ai/code) がこのリポジトリで作業する際のガイダンスを提供します。

## Claudeとの対話のガイドライン

- 常に日本語で会話してください
- ブランチの作成、コミット、リモートへの `git push`、Pull Request の作成・更新は、ユーザーからの明示的な指示があるまで行わないでください。ファイル編集、テスト実行、Linterの実行などのローカル作業は指示の範囲内で進めてかまいません
- ユーザーから明示的に質問を求められた場合 (「質問してください」「共通の理解に達するまで質問してください」など) は、Auto モードで動作している場合であっても質問を優先してください。推奨回答を添えて一度に一つずつ質問し、コードベースで調べられることは調べた上で、未解決の疑問点を必ず確認してから作業を開始してください

## プロジェクト概要

Mewst はマイクロブログサービスです。
ユーザーは短文のポストを作成することができ、ユーザーをフォローするとタイムラインにポストが時系列で表示されます。

## モノレポ構造

このリポジトリは、Go 版と Rails 版の 2 つのサブプロジェクトをモノレポとして管理しています：

```
/workspace/
├── go/                  # Go 版の実装 (段階的に機能を移行中)
├── rails/               # Rails 版の実装 (既存の本番システム)
├── caddy/               # リバースプロキシ設定
├── docs/                # Mewst 固有のドキュメント (仕様書、作業計画書など)
├── .claude/
│   ├── rules/
│   │   ├── korylus/     # Korylus 共通ガイドライン (korylus-guidelines をマウント)
│   │   └── mewst/       # Mewst 固有ガイドライン (Git 管理)
│   └── skills/
│       ├── korylus-*/   # Korylus 共通スキル (korylus-guidelines をマウント)
│       └── mewst-*/     # Mewst 固有スキル (Git 管理)
├── .github/             # 共通の CI/CD 設定
├── Dockerfile.dev       # 統合開発コンテナの Dockerfile
├── docker-compose.yml   # Docker Compose 設定
└── CLAUDE.md            # このファイル (プロジェクト全体のガイド)
```

## 主要な技術スタックのバージョン

Mewst で使用している主要なランタイム・ミドルウェアのバージョンは以下の通りです。

| 項目       | バージョン | 備考                                              |
| ---------- | ---------- | ------------------------------------------------- |
| Go         | 1.25.4     | Go 版で使用 (`/workspace/go/`)                    |
| Ruby       | 3.3.6      | Rails 版で使用 (`/workspace/rails/`)              |
| Rails      | 7.1.x      | Rails 版で使用 (`/workspace/rails/`)              |
| PostgreSQL | 18.3       | Go 版と Rails 版で共有 (詳細は「共通インフラ」節) |

なお、Korylus 共通ガイドライン (`korylus-guidelines`) はバージョン中立な記述としており、具体的なバージョンは本ファイルで管理しています。

## Rails から Go への移行について

現在、既存の Rails 実装の Mewst を Go で段階的に再実装するプロジェクトが進行中です。

### 移行の基本方針

- **既存 DB をそのまま使用**: Rails 側で管理されている PostgreSQL データベースを共有
- **段階的移行**: Rails と Go が同一の DB とセッションストアを共有し、段階的に機能を移行
- **データマイグレーション不要**: DB スキーマは既存のものを使用し、データ移行は行わない
- **共通インフラの継続利用**: PostgreSQL などの共通インフラは Go 版移行後も継続して使用

### Rails 側のソースコード

Rails 版のソースコードは `/workspace/rails/` 配下に格納されています：

```
/workspace/rails/
├── app/controllers/ # コントローラー
├── app/records/ # ActiveRecordモデル
├── app/use_cases/ # ユースケース (ビジネスロジック)
├── app/views/ # ビューテンプレート
├── config/routes.rb # ルーティング定義
└── db/structure.sql # DBスキーマ
```

Go 版を実装する際は、Rails 版のコードを参考にすることで既存の仕様を理解できます。

## フィーチャーフラグによる開発

Mewst ではフィーチャーブランチではなく **フィーチャーフラグ** を使って機能の公開を制御しています。基本方針・運用ルールは [@.claude/rules/korylus/common.md](/workspace/.claude/rules/korylus/common.md) の「フィーチャーフラグによる開発」セクションを参照してください。

Mewst で利用しているフラグの種類は次の 2 つです。

- **ルーティング制御 (Rails→Go 移行)**: Go 版のリバースプロキシミドルウェア ([@go/internal/middleware/reverse_proxy.go](/workspace/go/internal/middleware/reverse_proxy.go)) で、Go 版で処理するパスをホワイトリスト (`goHandledPaths`) に追加することで切り替えます。Go 版で未実装のパスは自動的に Rails 版にプロキシされます。新規機能を段階的にリリースする際もこの仕組みで制御します
- **アプリケーション内制御**: 現状 Mewst にはアプリケーション内フィーチャーフラグの専用機構 (Flipper / GrowthBook など) は **未導入** です。必要になった時点で導入方針を検討します

## 共通インフラ

### データベース (PostgreSQL)

- **バージョン**: PostgreSQL 18.3
- **共有方針**: Rails 版と Go 版で同一のデータベースを共有
- **開発環境**: Docker Compose で管理 (ポート: 4104)
- **データベース名**:
  - 開発: `mewst_development`
  - テスト: `mewst_test`

### セッションストア (PostgreSQL)

- **ストレージ**: PostgreSQL の `sessions` テーブルを使用
- **Rails 版**: ActiveRecord SessionStore を使用
  - 各リクエストで `updated_at` カラムを自動更新
  - セッションの有効期限: 30 日
- **Go 版**: 同じ `sessions` テーブルを共有
  - 認証ミドルウェアで `updated_at` カラムを更新
  - Rails 版と完全に互換性のあるセッション管理を実現
- **セッションクリーンアップ**: 毎日 19:00 に `rake session:sweep` タスクが実行され、30 日以上前のセッションを自動削除
- **共有方針**: Rails 版と Go 版で同一のセッションストアを共有 (段階的移行を実現)

## 開発環境のセットアップ

### 前提条件

- Docker 及び Docker Compose がインストール済み

### セットアップ手順

1. **リポジトリのクローン**

```sh
git clone git@github.com:mewstcom/mewst.git
cd mewst
```

2. **Docker Compose の起動**

```sh
docker compose up
```

3. **開発コンテナへの接続**

Go 版と Rails 版は統合された単一の開発コンテナ (`app`) で動作します。

```sh
docker compose exec app zsh
```

4. **各サブプロジェクトのセットアップ**

各サブプロジェクトの詳細なセットアップ手順は、以下のガイドラインを参照してください：

- Go 版: [@.claude/rules/korylus/go-development.md](/workspace/.claude/rules/korylus/go-development.md)
- Rails 版: [@.claude/rules/korylus/rails-common.md](/workspace/.claude/rules/korylus/rails-common.md)

### 環境変数の設定

各サブプロジェクトで `.env` ファイルを作成し、必要な環境変数を設定します。詳細は上記のガイドラインを参照してください。

### 開発サーバーの起動

プロジェクトルートで以下のコマンドを実行すると、Go 版・Rails 版の全サービスを一括で起動できます：

```sh
make dev
```

このコマンドは [hivemind](https://github.com/DarthSim/hivemind) を使用して `Procfile.dev` に定義された以下のプロセスを並行起動します：

| プロセス       | 内容                                        |
| -------------- | ------------------------------------------- |
| `go-server`    | Go 版サーバー (air によるホットリロード)    |
| `go-assets`    | Go 版フロントエンドアセットの監視・再ビルド |
| `rails-server` | Rails 版サーバー                            |
| `rails-css`    | Rails 版 CSS の監視・再ビルド               |
| `rails-js`     | Rails 版 JavaScript の監視・再ビルド        |

## ドキュメント

各機能の仕様は `docs/specs/` ディレクトリで管理しています。システムの現在の状態を理解するには、まず仕様書を参照してください。

- [@docs/README.md](/workspace/docs/README.md) - ドキュメント管理のガイド
- [@docs/specs/](/workspace/docs/specs/) - 各機能の仕様書

## 参照するガイドライン

Claude Code は `.claude/rules/` 配下のガイドラインを自動で読み込むため、通常は特に意識せず書いて OK。Korylus 共通ガイドラインの実体は `korylus-guidelines` リポジトリにあり、Docker Compose で `/korylus-guidelines/.claude/rules/korylus/` を `.claude/rules/korylus/` にマウントすることで参照しています (スキルも `/korylus-guidelines/.claude/skills/korylus-*` を `.claude/skills/korylus-*` にマウント)。

- **Korylus 共通**: `.claude/rules/korylus/common.md` / `.claude/rules/korylus/guidelines-authoring.md`
- **Go 版**: `.claude/rules/korylus/go-*.md` (coding, architecture, common, development, handler, usecase, testing, validation, security, templ, i18n)
- **Rails 版**: `.claude/rules/korylus/rails-*.md` (common, architecture, testing, security)

`.claude/rules/korylus/` 配下のファイルを編集すると、マウント元である `korylus-guidelines` リポジトリのファイルが直接更新されます。共通ガイドラインの修正は `korylus-guidelines` 側でコミットしてください。

## Mewst 固有のガイドライン

マウントされる共通ガイドライン (`.claude/rules/korylus/`) に加えて、Mewst プロジェクト固有の規約を本セクションに記述する。Korylus の他プロダクトには適用されない、Mewst 独自のドメイン規約・セキュリティ規約を扱う。

当面は本ファイルに直接記述し、記述量が増えてきたタイミングでトピックごとに `.claude/rules/mewst/{topic}.md` に切り出す (Mewst リポジトリで Git 管理)。切り出す際は YAML フロントマターの `paths:` で自動読み込みの対象範囲を指定する。

## レビュー時に参照するガイドライン

コードレビュー時に参照するガイドラインドキュメントの一覧です。変更されたファイルの種類に応じて、該当するガイドラインをチェックしてください。

### 共通ガイドライン

- [@CLAUDE.md](/workspace/CLAUDE.md) - プロジェクト全体のガイド
  - コミットメッセージのガイドライン
  - コメントのガイドライン
  - Pull Request のガイドライン
- [@.claude/rules/korylus/common.md](/workspace/.claude/rules/korylus/common.md) - Korylus プロダクト共通の開発ガイドライン (設計原則、PR・コミットのガイドラインなど)

### Go 版ガイドライン

- [@.claude/rules/korylus/go-common.md](/workspace/.claude/rules/korylus/go-common.md) - Go 版開発ガイド (プロジェクト構造、技術スタック、開発コマンド)
- [@.claude/rules/korylus/go-coding.md](/workspace/.claude/rules/korylus/go-coding.md) - コーディング規約 (インデント、フォーマット、コメント、log/slog)
- [@.claude/rules/korylus/go-architecture.md](/workspace/.claude/rules/korylus/go-architecture.md) - アーキテクチャガイド
  - 3 層アーキテクチャの依存関係ルール
  - Usecase、Repository の使い分け
- [@.claude/rules/korylus/go-usecase.md](/workspace/.claude/rules/korylus/go-usecase.md) - UseCase ガイド
  - 3 種類の分類 (読み取り / 書き込み / オーケストレーション)
  - UseCase 内の処理順序 (5 ステップ)
  - 書き込み UseCase の 2 つのルール
  - Validator のデータ取得パターン
- [@.claude/rules/korylus/go-handler.md](/workspace/.claude/rules/korylus/go-handler.md) - HTTP ハンドラーガイドライン
  - ディレクトリ構造
  - 標準ファイル名
  - 依存性注入
- [@.claude/rules/korylus/go-validation.md](/workspace/.claude/rules/korylus/go-validation.md) - バリデーションガイド
  - バリデーションの分類
  - 状態バリデーションの配置基準
- [@.claude/rules/korylus/go-i18n.md](/workspace/.claude/rules/korylus/go-i18n.md) - 国際化ガイド
  - 翻訳ファイルの追加手順
  - 命名規則
- [@.claude/rules/korylus/go-security.md](/workspace/.claude/rules/korylus/go-security.md) - セキュリティガイドライン
  - CSRF 対策
  - XSS 対策
  - SQL インジェクション対策
  - 認証・認可
- [@.claude/rules/korylus/go-templ.md](/workspace/.claude/rules/korylus/go-templ.md) - templ テンプレートガイド
  - ファイル配置
  - 命名規則
  - コンポーネント化
- [@.claude/rules/korylus/go-testing.md](/workspace/.claude/rules/korylus/go-testing.md) - テスト戦略
- [@.claude/rules/korylus/go-development.md](/workspace/.claude/rules/korylus/go-development.md) - 開発環境ガイド (DB マイグレーション、golangci-lint など)

### Rails 版ガイドライン

- [@.claude/rules/korylus/rails-common.md](/workspace/.claude/rules/korylus/rails-common.md) - Rails 版開発ガイド (プロジェクト構造、コーディング規約、開発コマンド)
- [@.claude/rules/korylus/rails-architecture.md](/workspace/.claude/rules/korylus/rails-architecture.md) - アーキテクチャガイド
  - アーキテクチャパターン (Records、UseCase、ViewComponent)
  - クラス間の依存関係ルール
  - 命名規則
- [@.claude/rules/korylus/rails-testing.md](/workspace/.claude/rules/korylus/rails-testing.md) - テストガイド
  - RSpec コーディング規約
  - テスト戦略
- [@.claude/rules/korylus/rails-security.md](/workspace/.claude/rules/korylus/rails-security.md) - セキュリティガイドライン
  - CSRF 対策
  - XSS 対策
  - SQL インジェクション対策
  - 認証

## 設計原則

### シンプルさと一貫性

このプロジェクトでは**シンプルさ**と**一貫性**を最も重要な設計原則として位置づけています。

- **シンプルさ (YAGNI)**: 過度な抽象化を避け、必要十分な複雑さに留める。必要になったときに必要な機能だけを実装する
- **一貫性**: 同じ種類の処理は同じパターンで実装する。ケースバイケースの判断を減らし、ルールを統一する

### 判断コストをゼロにする

設計上のルールは「どこに書くべきか」「このケースはどうするか」という判断が不要になるように定める。

- ❌ 「場合によっては A、場合によっては B」→ 判断コストが発生する
- ✅ 「常に A」→ 判断コストがゼロ

例: バリデーションは常に専用のディレクトリに配置する (Go 版: `internal/validator/`、Rails 版: `app/validators/`)。形式チェックのみでも、DB を使った検証を含む場合でも同じ場所に統一する。

## 開発ワークフロー

### 実装時のガイドライン

**既存コードとの一貫性**:

実装を行う前に、コードベース内に類似の処理がないか確認してください。
類似処理が存在する場合は、そのパターンに従って実装することで、コードベース全体の一貫性を保ちます。

- **確認すべき点**:
  - 同様の機能を持つハンドラー、ユースケース、リポジトリの実装パターン
  - エラーハンドリングの方法
  - ログ出力のフォーマット
  - バリデーションの実装方法
  - テストの書き方

- **類似処理が見つかった場合**: そのパターンを踏襲して実装する
- **類似処理が見つからない場合**: 各サブプロジェクトのガイドラインに従って新しいパターンを作成する

### コミット前のチェック

各サブプロジェクトで実装を行った場合は、コミット前に以下を確認してください：

- コードフォーマット
- リント
- テスト

JavaScript、CSS、JSON、YAML、Markdown、TOML ファイルのフォーマットは、プロジェクトルートの Oxfmt で管理しています：

```sh
# フォーマットチェック
make fmt-check

# 自動フォーマット
make fmt
```

各サブプロジェクト固有のコマンド (Go の `make fmt` や Rails の `make fmt` など) は、`.claude/rules/korylus/go-common.md` および `.claude/rules/korylus/rails-common.md` を参照してください。

### 修正後のコミット

**重要**: バグ修正や機能実装を行った場合でも、Claude Code が自動的にコミットを作成しないでください。

- 修正が完了したら、コミット前のチェック (フォーマット、リント、テスト) を実行して CI が通ることを確認
- **コミットは `/commit` コマンドで行う**: ユーザーが差分を確認し、適切な粒度でコミットできるようにする
- コミットメッセージは[コミットメッセージのガイドライン](#コミットメッセージのガイドライン)に従って日本語で記述

### コミットメッセージのガイドライン

コミットメッセージは**日本語**で記述してください。

**フォーマット**:

```
<タイトル> (1行、簡潔に変更内容を要約)

<本文> (任意、変更の詳細や理由を説明)
```

**良い例**:

```
パスワードリセット機能を実装

- internal/handler/password_reset/にハンドラーを追加
- internal/usecase/reset_password.goにビジネスロジックを実装
- Resend APIを使用したメール送信機能を追加
- Cloudflare TurnstileによるBot対策を実装
```

```
ユーザー認証のバグを修正

セッションタイムアウト後にリダイレクトが正しく動作しない
問題を修正。
```

**悪い例**:

- ❌ `Update handler` (英語、内容が不明確)
- ❌ `fix` (何を修正したか不明)
- ❌ `WIP` (作業中のコミットは避ける)

**原則**:

- タイトルは変更内容を簡潔に表現する
- 必要に応じて本文で詳細を説明する
- 関連する Issue やPR がある場合は参照を含める

### コメントのガイドライン

このガイドラインは Go 版と Rails 版の両方に適用されます。

**良いコメント**：

- コードの**意図や理由**を説明する (「なぜこうしたか」)
- 将来の開発者が理解できる、文脈に依存しない内容
- 複雑なロジックや、一見不自然に見える実装の背景を説明する

**避けるべきコメント**：

- ❌ **実装の変遷を説明するコメント** (「以前は〜だった」「〜は削除した」など)
- ❌ **過去との比較** (「別途インストール不要になった」「〜を統合したため不要」など)
- ❌ **自明なことの説明** (コードを読めばわかること)
- ❌ **やり取りの文脈に依存するコメント** (PR レビューのコメントは PR に書く)

**原則**：

- **コメントはコードの「なぜ」を説明し、「何を」はコードに語らせる**
- git の履歴に残る情報 (過去の実装、変更の経緯) はコメントに書かない
- レビューコメントや議論の文脈に依存する内容は書かない

詳細な例については、各サブプロジェクトのコーディング規約を参照してください：

- Go 版: [@.claude/rules/korylus/go-coding.md](/workspace/.claude/rules/korylus/go-coding.md) の「コメントのガイドライン」セクション
- Rails 版: [@.claude/rules/korylus/rails-common.md](/workspace/.claude/rules/korylus/rails-common.md) の「コメントのガイドライン」セクション

### Pull Request のガイドライン

Pull Request を作成する際は、以下のルールを遵守してください：

#### サイズの制限

- **変更ファイル数**: 20 ファイル以下
- **実装コードの行数**: 300 行以下を目安 (追加・削除行の合計)
- **テストコードの行数**: 制限なし (必要な分だけ追加して OK)

#### 実装とテストのセット化

- **必須**: 実装コードとそのテストコードは同じ PR に含める
- 新機能や修正を行う場合は、必ず対応するテストを追加・更新する
- テストがない実装は原則としてマージしない
- **テストは品質保証のために必要な分だけ書く**: 行数を気にせず、正常系・異常系・境界値などを網羅する

#### PR を小さく保つ理由

- レビュアーの負担を軽減し、レビューの質を向上させる
- バグの混入リスクを最小化する
- 問題が発生した場合のロールバックを容易にする
- CI/CD パイプラインの実行時間を短縮する

#### 大きな変更が必要な場合

機能が大きくなる場合は、以下のように分割してください：

1. **段階的な実装**: 機能を複数のステップに分割し、それぞれ独立した PR として作成
2. **リファクタリングの分離**: リファクタリングと新機能追加を別々の PR に分ける

#### 例外

以下の場合は制限を超えることが許容されます：

- 自動生成されたファイル (マイグレーション、スキーマなど)
- 広範囲に影響する命名変更やリファクタリング
- ただし、これらの場合でも可能な限り分割を検討してください

#### 重要な原則

**品質優先**: 上記の行数制限はあくまで**目安**です。以下の点を優先してください：

- **テストの完全性**: 実装にはテストを必ず含める。行数制限のためにテストを省略しない
- **コードの完全性**: 機能を中途半端な状態で分割しない。動作する最小単位で PR を作成する
- **可読性**: 無理に行数を減らすためにコードの可読性を犠牲にしない

行数制限を超えても、以下を満たしていれば問題ありません：

- 実装コードとテストコードの両方が含まれている
- コードレビューが可能な範囲 (目安: 1 ファイルあたり 500 行以下)
- PR の目的が明確で、1 つの機能や修正に集中している

**判断基準**: 「行数を守ること」よりも「きちんと実装すること」を優先してください。

## CI/CD

このモノレポの CI/CD 設定は`.github/workflows/`ディレクトリに配置されています：

- `go-ci.yml`: Go 版の CI (lint、test、build)
- `rails-ci.yml`: Rails 版の CI (zeitwerk、sorbet、standard、erb_lint、eslint、rspec)
- `fmt-ci.yml`: プロジェクト全体のフォーマットチェック (Oxfmt)

各 CI は対応するファイルが変更されたときに実行されます。

## トラブルシューティング

### データベース接続エラー

- PostgreSQL コンテナが起動しているか確認: `docker compose ps`
- ポートが正しいか確認: 開発環境は 4104

### その他の問題

各サブプロジェクト固有の問題については、`.claude/rules/korylus/go-common.md` および `.claude/rules/korylus/rails-common.md` を参照してください。
