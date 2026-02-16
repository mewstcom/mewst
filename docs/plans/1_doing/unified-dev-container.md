# 開発コンテナ統合 作業計画書

<!--
このテンプレートの使い方:
1. このファイルを `docs/plans/2_todo/` ディレクトリにコピー
   例: cp docs/plans/template.md docs/plans/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**作業計画書の性質**:
- 作業計画書は「何をどう変えるか」という変更内容を記述するドキュメントです
- 新しい機能の場合は、概要・要件・設計もこのドキュメントに記述します
- 現在のシステムの状態は `docs/specs/` の仕様書に記述されています
- タスク完了後は、仕様書を新しい状態に更新してください（設計判断や採用しなかった方針も含める）

**仕様書との関係**:
- 新しい機能の場合: タスク完了後に `docs/specs/` に仕様書を作成する
- 既存機能の変更の場合: 「仕様書」セクションに対応する仕様書へのリンクを記載し、タスク完了後に仕様書を更新する

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 仕様書

<!--
- 既存機能を変更する場合: 変更対象の仕様書へのリンクを記載してください
- 新しい機能の場合: タスク完了後に作成予定の仕様書のパスを記載してください
-->

- 新規作成: `docs/specs/dev-container.md`（タスク完了後に作成）

## 概要

<!--
ガイドライン:
- この機能が「何であるか」「なぜ必要か」を簡潔に説明
- 2-3段落程度で簡潔に
- 既存機能の変更の場合は、変更の背景と目的を記述
-->

現在、Go 版と Rails 版の開発環境は別々の Docker コンテナ（`go-app`, `rails-app`）で動作しています。これは両者が異なるバージョンの Node.js（Go: v24.11.1、Rails: v20.11.1）を使用しているためです。しかし、Go 版と Rails 版を同時に編集したい場面があり、コンテナ間の切り替えが不便です。

本設計では、開発ツールバージョンマネージャー [mise](https://mise.jdx.dev/) を導入することで、1 つの Docker コンテナ内で Go 版と Rails 版の両方を開発可能にします。

**目的**:

- Go 版と Rails 版を同一コンテナで開発可能にし、開発体験を向上させる
- Claude Code から両方のコードベースを同時に操作可能にする

**背景**:

- Go 版と Rails 版が同一の PostgreSQL データベースを共有し、段階的に機能移行を進めている
- 両方のコードを同時に参照・編集する場面が増えている
- コンテナ間の切り替えが開発効率を低下させている

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

<!--
「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
箇条書きで簡潔に
-->

- 1 つの Docker コンテナ内で Go 版の開発コマンド（`make test`, `make lint`, `air` 等）がすべて動作する
- 1 つの Docker コンテナ内で Rails 版の開発コマンド（`make test`, `make lint`, `bin/check` 等）がすべて動作する
- Go 版のディレクトリ（`/workspace/go/`）では Node.js v24.11.1 が使用される
- Rails 版のディレクトリ（`/workspace/rails/`）では Node.js v20.11.1 が使用される
- 既存の `make` コマンドやワークフローが変更なく動作する

### 非機能要件

<!--
必要に応じて以下のような項目を追加してください：
- セキュリティ（認証、認可、暗号化、監査ログなど）
- パフォーマンス（応答時間、スループット、リソース使用量など）
- ユーザビリティ（UX）（使いやすさ、わかりやすさ、アクセシビリティなど）
- 可用性・信頼性（稼働率、障害時の挙動、エラーハンドリングなど）
- 保守性（テストのしやすさ、コードの読みやすさ、ドキュメントなど）

不要な場合はこのセクション全体を削除してください。
-->

- Docker イメージのビルド時間が大幅に増加しないこと（目安: 10 分以内）
- コンテナ起動後の開発体験が現状と同等以上であること

## 実装ガイドラインの参照

<!--
**重要**: 設計を行う前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
設計の段階でガイドラインに準拠していることを確認してください。
-->

本作業計画書は Docker 環境の設定変更のため、プラットフォーム固有のガイドラインは適用されません。

## 設計

<!--
ガイドライン:
- 技術的な実装の設計を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - UI設計（画面構成、ユーザーフローなど）
  - セキュリティ設計（認証・認可、トークン管理など）
  - コード設計（パッケージ構成、主要な構造体など）

**重要: 設計は実装中に更新する**:
- 作業計画書内の設計は初期の方針であり、完璧ではない
- 実装中により良いアプローチが見つかった場合は、設計を積極的に更新する
- 設計に固執して実装の質を下げるよりも、実装で得た知見を設計に反映する方が重要
- 変更した場合は「採用しなかった方針」セクションに変更前の方針と変更理由を記録する
-->

### 現状の構成

```
docker-compose.yml
├── go-app      (golang:1.25.4-trixie ベース)
│   ├── Go 1.25.4
│   ├── Node.js 24.11.1 (NodeSource)
│   ├── pnpm 10.24.0
│   ├── golangci-lint, dbmate, sqlc, air
│   └── 共通ツール (postgresql-client-16, 1Password CLI, Claude Code, zsh等)
│
├── rails-app   (ruby:3.3.6-slim-bookworm ベース)
│   ├── Ruby 3.3.6
│   ├── Node.js 20.11.1 (NodeSource)
│   ├── Yarn 1.22.19
│   ├── Bundler 2.5.9, ImageMagick (libvips42)
│   └── 共通ツール (postgresql-client-16, 1Password CLI, Claude Code, zsh等)
│
├── caddy       (リバースプロキシ)
├── postgresql  (共有DB: PostgreSQL 16.2)
└── chrome      (E2Eテスト用)
```

### 統合後の構成

```
docker-compose.yml
├── app         (debian:trixie ベース、統合コンテナ)
│   ├── mise (開発ツールバージョンマネージャー)
│   │   ├── Go 1.25.4
│   │   ├── Ruby 3.3.6
│   │   ├── Node.js 24.11.1 + pnpm 10.24.0 → /workspace/go/ で使用
│   │   └── Node.js 20.11.1 + Yarn 1.22.19 → /workspace/rails/ で使用
│   ├── Go ツール: golangci-lint, dbmate, sqlc, air
│   ├── Rails ツール: Bundler 2.5.9, libvips42
│   └── 共通ツール (postgresql-client-16, 1Password CLI, Claude Code, zsh等)
│
├── caddy       (変更なし)
├── postgresql  (変更なし)
└── chrome      (変更なし)
```

### 開発ツールバージョン管理: mise

[mise](https://mise.jdx.dev/) を使用して Go、Ruby、Node.js、pnpm、Yarn のすべての開発ツールバージョンを統一管理します。

**mise を選択した理由**:

- Go、Ruby、Node.js、pnpm、Yarn 等を統一的に管理できるポリグロットなバージョンマネージャー
- `mise.toml` ファイルによるディレクトリ単位のバージョン自動切り替え
- asdf 互換の `.tool-versions` もサポート
- 環境変数の管理やタスクランナー機能も備えており、将来的に活用の幅が広い
- Docker 環境での利用が公式にサポートされている（shims 方式）
- 個別のインストール手順（Go tarball、ruby-install 等）が不要になり、Dockerfile がシンプルになる

**mise で管理するツール**:

| ツール  | バージョン | 用途                    |
| ------- | ---------- | ----------------------- |
| Go      | 1.25.4     | Go 版の開発             |
| Ruby    | 3.3.6      | Rails 版の開発          |
| Node.js | 24.11.1    | Go 版のフロントエンド   |
| Node.js | 20.11.1    | Rails 版のフロントエンド |
| pnpm    | 10.24.0    | Go 版のパッケージ管理   |
| Yarn    | 1.22.19    | Rails 版のパッケージ管理 |

**動作の仕組み**:

1. ルートの `mise.toml` で Go、Ruby のバージョンを指定（プロジェクト全体で共通）
2. 各サブプロジェクトの `mise.toml` で Node.js、pnpm/Yarn のバージョンを指定（ディレクトリごとに異なる）
3. Docker 環境では shims 方式を使用し、`/mise/shims` を PATH に含めることでツールバージョンを自動切り替え
4. `make` コマンド等のスクリプトからも `mise exec -- <command>` で明示的にバージョンを指定可能

```toml
# /workspace/mise.toml（プロジェクト全体で共通のツール）
[tools]
go = "1.25.4"
ruby = "3.3.6"

# /workspace/go/mise.toml（Go版固有）
[tools]
node = "24.11.1"
pnpm = "10.24.0"

# /workspace/rails/mise.toml（Rails版固有）
[tools]
node = "20.11.1"
yarn = "1.22.19"
```

### パッケージマネージャの管理

Go 版は pnpm、Rails 版は Yarn を使用しており、それぞれ mise 経由でインストール・管理します。各サブプロジェクトの `mise.toml` でバージョンを指定することで、ディレクトリに応じて自動的に正しいバージョンが使用されます。

corepack は使用しません。mise で pnpm/Yarn を直接管理することで、Node.js とパッケージマネージャのバージョン管理が同じ仕組みに統一されます。

### ベースイメージの選定

`debian:trixie` をベースイメージとして使用し、Go・Ruby・Node.js はすべて mise 経由でインストールします。

**理由**:

- 現在の Go 版 Dockerfile が `golang:1.25.4-trixie`（Debian Trixie）ベースである
- `golang:` や `ruby:` の公式イメージをベースにすると、もう一方の言語を追加インストールする際に複雑になる
- ニュートラルな Debian ベースに mise を入れ、すべての言語ランタイムを mise で管理するのが最もシンプル

**Go、Ruby、Node.js、pnpm、Yarn のインストール方法**: すべて mise 経由でインストール。個別のインストール手順（Go tarball、ruby-install、corepack 等）は不要

### Dockerfile.dev の設計

```dockerfile
FROM debian:trixie

ARG USER_ID=1000
ARG GROUP_ID=1000
ARG USERNAME=developer

# --- システムパッケージ ---
RUN <<EOF
apt update
apt dist-upgrade -yq

PACKAGES=$(cat <<'PKGLIST' | sed 's/#.*//'
  build-essential
  ca-certificates           # HTTPS接続に必要
  curl
  file                      # Shrineによる画像アップロードに必要 (Rails)
  fzf                       # Claude Codeで使用
  git
  gnupg                     # PostgreSQLのGPGキーとaptリポジトリに必要
  jq                        # Claude Codeで使用
  libffi-dev                # Ruby拡張のビルドに必要
  libglib2.0-0              # libvipsの依存パッケージ (Rails)
  libreadline-dev           # Rubyのreadline拡張に必要
  libssl-dev                # SSL関連のビルドに必要
  libvips42                 # 画像処理ライブラリ (Rails)
  libyaml-dev               # psych gemのインストールに必要 (Rails)
  lsb-release               # PostgreSQLのaptリポジトリ設定に必要
  nano                      # Claude Codeで使用
  ripgrep                   # Claude Codeで使用
  sudo
  tree                      # Claude Codeで使用
  unzip
  vim                       # Claude Codeで使用
  zsh
PKGLIST
)

apt install -y --no-install-recommends $PACKAGES
rm -rf /var/lib/apt/lists/*
EOF

# --- PostgreSQL クライアント ---
RUN curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
      | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg && \
    echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
      > /etc/apt/sources.list.d/pgdg.list && \
    apt update && apt install -y libpq-dev postgresql-client-16 && \
    rm -rf /var/lib/apt/lists/*

# --- mise (開発ツールバージョンマネージャー) ---
# Docker環境向けの推奨設定: shims方式を使用
ENV MISE_DATA_DIR="/mise"
ENV MISE_CONFIG_DIR="/mise"
ENV MISE_CACHE_DIR="/mise/cache"
ENV MISE_INSTALL_PATH="/usr/local/bin/mise"
ENV PATH="/mise/shims:${PATH}"
RUN curl https://mise.run | sh

# Go、Ruby、Node.js、pnpm、Yarnをmise経由でインストール
RUN mise install go@1.25.4 && \
    mise install ruby@3.3.6 && \
    mise install node@24.11.1 && \
    mise install node@20.11.1 && \
    mise install pnpm@10.24.0 && \
    mise install yarn@1.22.19

# グローバルデフォルトを設定
RUN mise use --global go@1.25.4 && \
    mise use --global ruby@3.3.6 && \
    mise use --global node@24.11.1 && \
    mise use --global pnpm@10.24.0

# --- dbmate ---
RUN curl -fsSL -o /usr/local/bin/dbmate \
      https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 && \
    chmod +x /usr/local/bin/dbmate

# --- golangci-lint ---
# mise管理のGoのbinディレクトリにインストール
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
      | sh -s -- -b /usr/local/bin v2.6.2

# --- ユーザー作成 ---
RUN groupadd -g ${GROUP_ID} ${USERNAME} && \
    useradd -u ${USER_ID} -g ${GROUP_ID} -m -s /bin/zsh ${USERNAME} && \
    echo "${USERNAME} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# --- Go キャッシュディレクトリ ---
RUN mkdir -p /go/.cache/go-build /go/pkg && \
    chown -R ${USER_ID}:${GROUP_ID} /go

# --- mise のデータディレクトリの権限設定 ---
RUN chown -R ${USER_ID}:${GROUP_ID} /mise

WORKDIR /workspace
USER ${USERNAME}

# --- mise シェル設定 ---
RUN echo 'eval "$(mise activate zsh)"' >> /home/${USERNAME}/.zshrc

# --- 1Password CLI ---
COPY --from=1password/op:2 /usr/local/bin/op /usr/local/bin/op

# --- Claude Code ---
RUN curl -fsSL https://claude.ai/install.sh | bash

# --- Go ツール ---
ENV GOCACHE=/go/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod
ENV GOPATH=/go
ENV PATH="${GOPATH}/bin:${PATH}"
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 && \
    go install github.com/air-verse/air@latest

# --- Ruby ツール ---
RUN gem install bundler -v 2.5.9

# --- シェル設定 ---
SHELL ["/bin/zsh", "-c"]
ENV SHELL=/bin/zsh

# --- Git設定 ---
RUN git config --global user.email "me@shimba.co" && \
    git config --global user.name "Koji Shimba"

EXPOSE 8080 3000

CMD ["/bin/zsh"]
```

### docker-compose.yml の変更

```yaml
services:
  app:
    build:
      context: .
      dockerfile: ./Dockerfile.dev
    depends_on:
      - caddy
      - postgresql
    volumes:
      - .:/workspace
      - op-config:/home/developer/.config/op
      - go-mod-cache:/go/pkg/mod
      - go-build-cache:/go/.cache/go-build
      - app-gems-data:/mise/installs/ruby/3.3.6/lib/ruby/gems/3.3.0
    ports:
      - "4100:8080" # Go
      - "4101:3000" # Rails
    stdin_open: true
    tty: true
    working_dir: /workspace
    environment:
      - BINDING=0.0.0.0
      - OP_SERVICE_ACCOUNT_TOKEN=${OP_SERVICE_ACCOUNT_TOKEN:-}

  caddy:
    image: caddy:2-alpine
    ports:
      - "4103:8080"
    volumes:
      - ./caddy/Caddyfile:/etc/caddy/Caddyfile:ro

  postgresql:
    image: postgres:16.2
    ports:
      - "4104:5432"
    volumes:
      - postgresql16_data:/var/lib/postgresql/data:delegated
    environment:
      POSTGRES_HOST_AUTH_METHOD: trust

  chrome:
    image: browserless/chrome:1.61.1-chrome-stable
    ports:
      - "4105:3333"
    environment:
      PORT: 3333
      CONNECTION_TIMEOUT: 600000

volumes:
  app-gems-data:
  go-build-cache:
  go-mod-cache:
  op-config:
  postgresql16_data:
```

### mise.toml ファイルの追加

ルートとサブプロジェクトに `mise.toml` ファイルを配置し、ツールバージョンを指定します。

```toml
# /workspace/mise.toml（プロジェクト全体で共通のツール）
[tools]
go = "1.25.4"
ruby = "3.3.6"

# /workspace/go/mise.toml（Go版固有）
[tools]
node = "24.11.1"
pnpm = "10.24.0"

# /workspace/rails/mise.toml（Rails版固有）
[tools]
node = "20.11.1"
yarn = "1.22.19"
```

mise は階層的な設定を持ち、カレントディレクトリから親ディレクトリへと`mise.toml`を探索します。設定はマージされるため:

- `/workspace/go/` 内では Go 1.25.4 + Ruby 3.3.6（親から継承）+ Node.js 24.11.1 + pnpm 10.24.0（自身の設定）
- `/workspace/rails/` 内では Go 1.25.4 + Ruby 3.3.6（親から継承）+ Node.js 20.11.1 + Yarn 1.22.19（自身の設定）

バージョン更新時は `mise.toml` を変更するだけで済み、Dockerfile の再ビルドは不要です（新しいバージョンは `mise install` で追加）。

### Makefile への影響

各サブプロジェクトの `Makefile` は基本的に変更不要です。mise の shims 方式により、`/mise/shims` が PATH に含まれていれば、カレントディレクトリの `mise.toml` に基づいて自動的に正しいバージョンの `go`、`ruby`、`node`、`pnpm`、`yarn` 等が使用されます。

`make` コマンドはサブシェルで実行されますが、shims 方式ではシェル初期化が不要なため、Makefile 内でコマンドを呼び出すだけで正しいバージョンが使用されます。

もし明示的にバージョンを指定する必要がある場合は、`mise exec` を使用できます:

```makefile
# 明示的にツールバージョンを指定する場合（通常は不要）
build:
	mise exec -- pnpm build
```

具体的な対応はタスク実装時に検証します。

### 移行の注意点

- **既存の Docker ボリューム**: `go-mod-cache`, `go-build-cache` はそのまま利用可能
- **Gem のインストール先**: mise 管理の Ruby はシステムとは異なるパスにインストールされるため、Gem のインストール先パスが変わる。`app-gems-data` ボリュームのマウントパスを調整する
- **Go の PATH**: mise 管理の Go は `/mise/installs/go/1.25.4` 配下にインストールされる。`GOROOT` の設定や `go install` でインストールされるバイナリのパスに注意する
- **ポート**: Go（8080）と Rails（3000）の両方を公開
- **旧コンテナの削除**: 移行後、`go-app` と `rails-app` サービスは削除

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録
- タスク完了後、この内容は `specs/` の仕様書にも転記する
- 該当がない場合は「なし」と記載
-->

なし

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- チェックボックスで進捗を管理
- **重要**: 1タスク = 1 Pull Request の粒度で作成してください
- **重要**: 各タスクには想定ファイル数と想定行数を明記してください（PRサイズの見積もりのため）
- 想定ファイル数は「実装」と「テスト」に分けて記載してください
- 想定行数も「実装」と「テスト」に分けて記載してください
- 依存関係を明確に
- Pull Requestのガイドラインは CLAUDE.md を参照（変更ファイル数20以下、変更行数300行以下）

タスク番号の付け方:
- 各タスクには階層的な番号を付与します（例: 1-1, 1-2, 2-1, 2-2）
- フォーマット: **フェーズ番号-タスク番号**: タスク名
- **フェーズ番号は半角英数字とハイフンのみで表記**してください（ブランチ名に使用するため）
  - 例: フェーズ 1, フェーズ 2, フェーズ 5a（フェーズ 5 と 6 の間に追加する場合）
  - NG: フェーズ 5.5（ドットは使用不可）
- タスクの前に別のタスクを追加する場合は、サブ番号を使用します
  - 例: タスク 2-1 の前にタスクを追加する場合 → 2-0
  - 例: タスク 2-0 の前にタスクを追加する場合 → 2-0-1
- この番号はブランチ名の一部として使用されます（例: feature-1-1, feature-2-0）

プラットフォームプレフィックス:
- Go版またはRails版の修正を行うタスクには、タスク名の先頭にプラットフォームを示すプレフィックスを付けてください
- フォーマット: **フェーズ番号-タスク番号**: [Go] タスク名 または **フェーズ番号-タスク番号**: [Rails] タスク名
- Go版とRails版の両方を修正する場合は、別々のタスクに分けてください
- 例:
  - `- [ ] **1-1**: [Go] マイグレーション作成`
  - `- [ ] **1-2**: [Rails] モデルへのコールバック追加`
-->

### フェーズ 1: 統合 Dockerfile の作成と検証

<!--
例: インフラ準備、基本機能実装、セキュリティ機能など
各タスクは1つのPull Requestで完結する粒度で記述してください
各タスクには想定サイズを明記してください
Go版/Rails版の両方を修正する場合は別タスクに分けてください
-->

- [x] **1-1**: 統合 Dockerfile.dev を作成

  - ルートに `Dockerfile.dev` を新規作成（上記設計に基づく）
  - `mise.toml`（ルート）、`go/mise.toml`、`rails/mise.toml` を作成
  - `docker-compose.yml` を更新（`go-app` + `rails-app` → `app` に統合）
  - コンテナをビルドして起動できることを確認
  - **想定ファイル数**: 約 5 ファイル（実装 5 + テスト 0）
  - **想定行数**: 約 150 行（実装 150 行 + テスト 0 行）

### フェーズ 2: Go 版の動作確認と調整

- [ ] **2-1**: [Go] 統合コンテナでの Go 版動作確認と調整

  - `cd /workspace/go && make test` が通ることを確認
  - `cd /workspace/go && make lint` が通ることを確認
  - `cd /workspace/go && make fmt` が動作することを確認
  - Node.js/pnpm 関連のコマンド（`pnpm install`, `pnpm build`）が正しいバージョンで動作することを確認
  - 必要に応じて Makefile や設定ファイルを調整
  - **想定ファイル数**: 約 3 ファイル（実装 3 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

### フェーズ 3: Rails 版の動作確認と調整

- [ ] **3-1**: [Rails] 統合コンテナでの Rails 版動作確認と調整

  - `cd /workspace/rails && make test` が通ることを確認
  - `cd /workspace/rails && make lint` が通ることを確認
  - `cd /workspace/rails && bin/check` が動作することを確認
  - Node.js/Yarn 関連のコマンドが正しいバージョンで動作することを確認
  - Gem のインストール先パスが正しいことを確認
  - 必要に応じて Makefile や設定ファイルを調整
  - **想定ファイル数**: 約 3 ファイル（実装 3 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

### フェーズ 4: クリーンアップ

- [ ] **4-1**: 旧 Dockerfile の削除とドキュメント更新

  - `go/Dockerfile.dev` を削除
  - `rails/Dockerfile.dev` を削除
  - `CLAUDE.md` の Docker 関連セクションを更新
  - `go/CLAUDE.md` のコンテナ関連セクションを更新
  - `rails/CLAUDE.md` のコンテナ関連セクションを更新
  - **想定ファイル数**: 約 5 ファイル（実装 5 + テスト 0）
  - **想定行数**: 約 50 行（実装 50 行 + テスト 0 行）

### フェーズ 5: 仕様書への反映

<!--
**重要**: 実装完了後、必ず仕様書を作成・更新してください。
- 新しい機能の場合: `docs/specs/` に仕様書を新規作成する
- 既存機能の変更の場合: 対応する仕様書を最新の状態に更新する
- 概要・仕様・設計・採用しなかった方針を作業計画書から転記・整理する
-->

- [ ] **5-1**: 仕様書の作成・更新
  - `docs/specs/dev-container.md` に仕様書を作成
  - 作業計画書の概要・要件・設計・採用しなかった方針を仕様書に反映する

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **本番用 Dockerfile の統合**: 本設計は開発環境（`Dockerfile.dev`）のみを対象とする。本番用 Dockerfile は引き続き個別に管理する
- **CI/CD の変更**: GitHub Actions の CI 設定は現状のまま維持する
- **devcontainer の導入**: VS Code Dev Container 設定の追加は今回のスコープ外

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [mise - 開発ツールバージョンマネージャー](https://mise.jdx.dev/)
- [mise - Docker 環境での使い方](https://mise.jdx.dev/containers/)
- [Wikino 開発コンテナ統合 設計書](/wikino/docs/plans/3_done/202602/unified-dev-container.md)
