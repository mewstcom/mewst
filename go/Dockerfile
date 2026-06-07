# syntax=docker/dockerfile:1.7

# ========================================
# Build stage
# [Ja] ビルドステージ
# ========================================
FROM golang:1.26.4-alpine3.23 AS builder

# Set working directory
# [Ja] 作業ディレクトリを設定
WORKDIR /app

# Install system dependencies:
#   - git:       needed by `go mod download` for VCS-based modules
#   - curl:      used to download the pinned dbmate binary below
#   - libstdc++: required to run the Node binary copied from node:alpine below
#
# [Ja] システム依存関係のインストール:
#   - git:       go mod download で VCS ベースのモジュールを取得するため
#   - curl:      下で固定バージョンの dbmate バイナリを取得するため
#   - libstdc++: 下で node:alpine からコピーする Node バイナリを動作させるため
RUN apk add --no-cache \
    git \
    curl \
    libstdc++

# Copy Node.js from the official image so the build uses the same Node version
# as Dockerfile.dev (mise's node@24.11.1), instead of whatever Alpine's `apk
# add nodejs` happens to provide. The corepack symlink under /usr/local/bin is
# recreated because COPY dereferences single-file symlinks.
#
# [Ja] Dockerfile.dev (mise の node@24.11.1) と Node バージョンを揃えるため、
# `apk add nodejs` (Alpine 提供版) ではなく公式 Node イメージからコピーする。
# /usr/local/bin/corepack のシンボリックリンクは COPY で参照先実体化されるため作り直す。
COPY --from=node:24.11.1-alpine3.23 /usr/local/bin/node /usr/local/bin/node
COPY --from=node:24.11.1-alpine3.23 /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/corepack/dist/corepack.js /usr/local/bin/corepack

# Download dbmate binary (pinned for reproducibility, verified by SHA256)
# [Ja] dbmate バイナリのダウンロード (バージョンを固定して再現性を確保、SHA256 で検証)
ARG DBMATE_VERSION=v2.33.0
ARG DBMATE_SHA256=dfd4027141b6e4357158099bdfe6c02b286af70f8e49057937744cf80b38ea1c
RUN curl -fsSL -o /tmp/dbmate https://github.com/amacneil/dbmate/releases/download/${DBMATE_VERSION}/dbmate-linux-amd64 && \
    echo "${DBMATE_SHA256}  /tmp/dbmate" | sha256sum -c - && \
    mv /tmp/dbmate /usr/local/bin/dbmate && \
    chmod +x /usr/local/bin/dbmate

# Enable Corepack and activate pnpm (version must match package.json's packageManager field).
# [Ja] Corepack を有効化し pnpm を有効化 (バージョンは package.json の packageManager フィールドと一致させる)。
RUN corepack enable && corepack prepare pnpm@10.24.0 --activate

# Copy Go dependency files
# [Ja] Go 依存関係ファイルをコピー
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy pnpm dependency files
# [Ja] pnpm 依存関係ファイルをコピー
COPY package.json pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

# Install templ and goimports at the versions pinned in go.mod (via tools.go).
# goimports is required to merge the import blocks that templ generates: when
# a .templ source already imports `github.com/a-h/templ`, templ emits a
# second import block for its runtime, and only goimports collapses the two.
# `make templ-generate` runs the same pair locally, so we mirror it here.
#
# [Ja] tools.go で go.mod に固定されたバージョンの templ と goimports を
# インストール。goimports は templ が生成する import ブロックをマージするのに
# 必要: `.templ` 側で `github.com/a-h/templ` を import していると、templ は
# runtime 用に別の import ブロックを生成するため、goimports でまとめない
# 限り `templ` シンボルが二重宣言になる。ローカルの `make templ-generate` も
# 同じ 2 つを順に呼ぶため、本番 Dockerfile もそれに揃える。
COPY tools.go ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go install -mod=mod github.com/a-h/templ/cmd/templ && \
    go install -mod=mod golang.org/x/tools/cmd/goimports

# Copy source code
# [Ja] ソースコードをコピー
COPY . .

# Generate Go code from templ files, then run goimports to merge the
# duplicate import blocks that templ emits when a .templ source already
# imports `github.com/a-h/templ` (e.g. card.templ, container.templ).
# Committed *_templ.go files are kept out of the build context by
# `.dockerignore` (**/*_templ.go), so this RUN starts from a clean tree
# without an explicit pre-delete step. This pair of steps matches the
# `templ-generate` -> `templ-goimports` chain in go/Makefile so local and
# production builds run the exact same scope (-w internal/templates).
#
# [Ja] templ ファイルから Go コードを生成し、続いて goimports で
# `.templ` 側が `github.com/a-h/templ` を import しているコンポーネント
# (card.templ / container.templ など) で発生する import ブロック重複を
# マージする。コミット済みの *_templ.go は `.dockerignore` の
# `**/*_templ.go` でビルドコンテキストから除外されているため、明示的に
# 削除する手順は不要 (クリーンな状態から生成される)。go/Makefile の
# `templ-generate` -> `templ-goimports` チェーンと同じスコープ
# (-w internal/templates) を共有することで、ローカルと本番ビルドの
# 挙動を揃えている。
RUN templ generate && \
    goimports -w internal/templates

# Build frontend assets
# [Ja] フロントエンドアセットをビルド
RUN pnpm build

# Build Go binary
# [Ja] Go バイナリをビルド
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o bin/server cmd/server/main.go

# ========================================
# Runtime stage
# [Ja] 実行ステージ
# ========================================
FROM alpine:3.23

# Set working directory
# [Ja] 作業ディレクトリを設定
WORKDIR /app

# Install runtime system dependencies:
#   - ca-certificates: needed for HTTPS calls (e.g. Resend API)
#   - tzdata:          required by TZ=UTC handling below
#   - bash:            Dokku runs each Procfile process (web / release)
#                      through bash, so /bin/bash must exist in the runtime
#                      image. Without it, `release: dbmate ...` fails with
#                      `exec: "/bin/bash": no such file or directory` even
#                      though the Go server itself only needs /bin/sh.
#
# [Ja] 実行時の依存パッケージ:
#   - ca-certificates: HTTPS 通信 (Resend API など) に必要
#   - tzdata:          下の TZ=UTC 設定で参照される
#   - bash:            Dokku は Procfile の各プロセス (web / release) を
#                      bash 経由で起動するため、ランタイムに /bin/bash が
#                      必要。Go サーバー自体は /bin/sh で動くが、bash が
#                      ないと `release: dbmate ...` の起動時に
#                      `exec: "/bin/bash": no such file or directory`
#                      で失敗する。
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    bash

# Copy dbmate from build stage (pinned for reproducibility)
# [Ja] ビルドステージから dbmate をコピー (バージョンを固定して再現性を確保)
COPY --from=builder /usr/local/bin/dbmate /usr/local/bin/dbmate

# Copy required files from build stage.
# Migrations only: schema.sql is a dev-time snapshot, not consumed by `dbmate up`.
# [Ja] ビルドステージから必要なファイルをコピー。
# migrations のみ: schema.sql は開発時のスナップショットで `dbmate up` は使わない。
COPY --from=builder /app/bin/server /app/bin/server
COPY --from=builder /app/static /app/static
COPY --from=builder /app/db/migrations /app/db/migrations

# Set timezone to UTC
# [Ja] タイムゾーンを UTC に設定
ENV TZ=UTC

# Expose port (Dokku detects the listening port via this directive)
# [Ja] ポートを公開 (Dokku はこのディレクティブで待受ポートを検出する)
EXPOSE 8080

# Default command. Dokku prefers the Procfile's `web` entry over this CMD,
# so production deploys still go through Procfile. CMD exists so that
# `docker run <image>` works for local/CI smoke tests.
#
# [Ja] デフォルトの起動コマンド。Dokku は Procfile の `web` を優先するため、
# 本番デプロイは Procfile 経由のままになる。CMD は `docker run <image>` で
# ローカルや CI のスモークテストを行うために置く。
CMD ["/app/bin/server"]
