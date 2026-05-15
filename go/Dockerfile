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
    npm \
    curl

# Install pnpm
# [Ja] pnpm のインストール
RUN npm install -g pnpm@10.24.0

# Copy Go dependency files
# [Ja] Go 依存関係ファイルをコピー
COPY go.mod go.sum ./
RUN go mod download

# Copy pnpm dependency files
# [Ja] pnpm 依存関係ファイルをコピー
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# Install development tools defined in tools.go
# [Ja] tools.go で定義された開発ツールをインストール
COPY tools.go ./
RUN go install github.com/a-h/templ/cmd/templ && \
    go install golang.org/x/tools/cmd/goimports

# Copy source code
# [Ja] ソースコードをコピー
COPY . .

# Generate Go code from templ files
# [Ja] templ ファイルから Go コードを生成
RUN templ generate

# Organize import statements
# [Ja] import 文を整理
RUN goimports -w .

# Build frontend assets
# [Ja] フロントエンドアセットをビルド
RUN pnpm build

# Build Go binary
# [Ja] Go バイナリをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/server cmd/server/main.go

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
    postgresql-client \
    curl \
    bash

# Install dbmate (for migrations)
# [Ja] dbmate のインストール（マイグレーション用）
RUN curl -fsSL -o /usr/local/bin/dbmate https://github.com/amacneil/dbmate/releases/latest/download/dbmate-linux-amd64 && \
    chmod +x /usr/local/bin/dbmate

# Copy required files from build stage
# [Ja] ビルドステージから必要なファイルをコピー
COPY --from=builder /app/bin/server /app/bin/server
COPY --from=builder /app/static /app/static
COPY --from=builder /app/db /app/db
COPY --from=builder /app/internal/i18n/locales /app/internal/i18n/locales

# Set timezone to UTC
# [Ja] タイムゾーンを UTC に設定
ENV TZ=UTC

# Default listen port (Dokku maps it automatically)
# [Ja] 待受ポートの既定値（Dokku が自動的にマッピング）
ENV MEWST_PORT=8080

# Expose port
# [Ja] ポートを公開
EXPOSE 8080
