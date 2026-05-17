# syntax=docker/dockerfile:1.7

# ========================================
# Build stage
# [Ja] ビルドステージ
# ========================================
FROM golang:1.25.4-alpine AS builder

# Set working directory
# [Ja] 作業ディレクトリを設定
WORKDIR /app

# Install system dependencies
# [Ja] システム依存関係のインストール
RUN apk add --no-cache \
    git \
    nodejs \
    curl

# Download dbmate binary (pinned for reproducibility, verified by SHA256)
# [Ja] dbmate バイナリのダウンロード (バージョンを固定して再現性を確保、SHA256 で検証)
ARG DBMATE_VERSION=v2.33.0
ARG DBMATE_SHA256=dfd4027141b6e4357158099bdfe6c02b286af70f8e49057937744cf80b38ea1c
RUN curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/download/${DBMATE_VERSION}/dbmate-linux-amd64 && \
    echo "${DBMATE_SHA256}  /usr/local/bin/dbmate" | sha256sum -c - && \
    chmod +x /usr/local/bin/dbmate

# Enable Corepack and activate pnpm at the version pinned in package.json
# [Ja] Corepack を有効化し、package.json で固定された pnpm を有効化
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

# Install templ at the version pinned in go.mod (via tools.go)
# [Ja] tools.go で go.mod に固定されたバージョンの templ をインストール
COPY tools.go ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go install -mod=mod github.com/a-h/templ/cmd/templ

# Copy source code
# [Ja] ソースコードをコピー
COPY . .

# Generate Go code from templ files
# [Ja] templ ファイルから Go コードを生成
RUN templ generate

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
FROM alpine:3.21

# Set working directory
# [Ja] 作業ディレクトリを設定
WORKDIR /app

# Install system dependencies
# [Ja] システム依存関係のインストール
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    bash

# Copy dbmate from build stage (pinned for reproducibility)
# [Ja] ビルドステージから dbmate をコピー (バージョンを固定して再現性を確保)
COPY --from=builder /usr/local/bin/dbmate /usr/local/bin/dbmate

# Copy required files from build stage
# [Ja] ビルドステージから必要なファイルをコピー
COPY --from=builder /app/bin/server /app/bin/server
COPY --from=builder /app/static /app/static
COPY --from=builder /app/db /app/db
COPY --from=builder /app/internal/i18n/locales /app/internal/i18n/locales

# Set timezone to UTC
# [Ja] タイムゾーンを UTC に設定
ENV TZ=UTC

# Expose port (Dokku detects the listening port via this directive)
# [Ja] ポートを公開 (Dokku はこのディレクティブで待受ポートを検出する)
EXPOSE 8080
