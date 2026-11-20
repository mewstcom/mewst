# syntax=docker/dockerfile:1.7

# ========================================
# ビルドステージ
# ========================================
FROM golang:1.26.4-alpine3.23 AS builder

# 作業ディレクトリを設定
WORKDIR /app

# システム依存関係のインストール:
#   - git:       go mod downloadでVCSベースのモジュールを取得するため
#   - curl:      下で固定バージョンのdbmateバイナリを取得するため
#   - libstdc++: 下でnode:alpineからコピーするNodeバイナリを動作させるため
RUN apk add --no-cache \
    git \
    curl \
    libstdc++

# Dockerfile.dev (miseのnode@24.11.1) とNodeバージョンを揃えるため、
# `apk add nodejs` (Alpine提供版) ではなく公式Nodeイメージからコピーする。
# /usr/local/bin/corepackのシンボリックリンクはCOPYで参照先実体化されるため作り直す。
COPY --from=node:24.11.1-alpine3.23 /usr/local/bin/node /usr/local/bin/node
COPY --from=node:24.11.1-alpine3.23 /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/corepack/dist/corepack.js /usr/local/bin/corepack

# dbmateバイナリのダウンロード (バージョンを固定して再現性を確保、SHA256で検証)
ARG DBMATE_VERSION=v2.33.0
ARG DBMATE_SHA256=dfd4027141b6e4357158099bdfe6c02b286af70f8e49057937744cf80b38ea1c
RUN curl -fsSL -o /tmp/dbmate https://github.com/amacneil/dbmate/releases/download/${DBMATE_VERSION}/dbmate-linux-amd64 && \
    echo "${DBMATE_SHA256}  /tmp/dbmate" | sha256sum -c - && \
    mv /tmp/dbmate /usr/local/bin/dbmate && \
    chmod +x /usr/local/bin/dbmate

# Corepackを有効化しpnpmを有効化 (バージョンはpackage.jsonのpackageManagerフィールドと一致させる)。
RUN corepack enable && corepack prepare pnpm@10.24.0 --activate

# Go依存関係ファイルをコピー
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

# pnpm依存関係ファイルをコピー
COPY package.json pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile

# tools.goでgo.modに固定されたバージョンのtemplとgoimportsを
# インストール。goimportsはtemplが生成するimportブロックをマージするのに
# 必要: `.templ` 側で `github.com/a-h/templ` をimportしていると、templは
# runtime用に別のimportブロックを生成するため、goimportsでまとめない
# 限り `templ` シンボルが二重宣言になる。ローカルの `make templ-generate` も
# 同じ2つを順に呼ぶため、本番Dockerfileもそれに揃える。
COPY tools.go ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go install -mod=mod github.com/a-h/templ/cmd/templ && \
    go install -mod=mod golang.org/x/tools/cmd/goimports

# ソースコードをコピー
COPY . .

# templファイルからGoコードを生成し、続いてgoimportsで
# `.templ` 側が `github.com/a-h/templ` をimportしているコンポーネント
# (card.templ / container.templなど) で発生するimportブロック重複を
# マージする。コミット済みの *_templ.goは `.dockerignore` の
# `**/*_templ.go` でビルドコンテキストから除外されているため、明示的に
# 削除する手順は不要 (クリーンな状態から生成される)。go/Makefileの
# `templ-generate` -> `templ-goimports` チェーンと同じスコープ
# (-w internal/templates) を共有することで、ローカルと本番ビルドの
# 挙動を揃えている。
RUN templ generate && \
    goimports -w internal/templates

# フロントエンドアセットをビルド
RUN pnpm build

# Goバイナリをビルド
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o bin/mewst ./cmd/mewst

# ========================================
# 実行ステージ
# ========================================
FROM alpine:3.23

# 作業ディレクトリを設定
WORKDIR /app

# 実行時の依存パッケージ:
#   - ca-certificates: HTTPS通信 (Resend APIなど) に必要
#   - tzdata:          下のTZ=UTC設定で参照される
#   - bash:            DokkuはProcfileの各プロセス (web / release) を
#                      bash経由で起動するため、ランタイムに /bin/bashが
#                      必要。Goサーバー自体は /bin/shで動くが、bashが
#                      ないと `release: dbmate ...` の起動時に
#                      `exec: "/bin/bash": no such file or directory`
#                      で失敗する。
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    bash

# ビルドステージからdbmateをコピー (バージョンを固定して再現性を確保)
COPY --from=builder /usr/local/bin/dbmate /usr/local/bin/dbmate

# ビルドステージから必要なファイルをコピー。
# migrationsのみ: schema.sqlは開発時のスナップショットで `dbmate up` は使わない。
COPY --from=builder /app/bin/mewst /app/bin/mewst
COPY --from=builder /app/static /app/static
COPY --from=builder /app/db/migrations /app/db/migrations

# タイムゾーンをUTCに設定
ENV TZ=UTC

# ポートを公開 (Dokkuはこのディレクティブで待受ポートを検出する)
EXPOSE 8080

# デフォルトの起動コマンド。DokkuはProcfileの `web` を優先するため、
# 本番デプロイはProcfile経由のままになる。CMDは `docker run <image>` で
# ローカルやCIのスモークテストを行うために置く。
CMD ["/app/bin/mewst", "serve"]
