# op run ラッパー: 1Password CLI 経由で .env の環境変数を解決してコマンドを実行する
# CI環境ではopコマンドが利用できないため、環境変数はGitHub Actionsで直接設定する
OP_RUN_DEV = APP_ENV=dev op run --env-file=".env" --
ifeq ($(CI),true)
OP_RUN_TEST = APP_ENV=test
else
OP_RUN_TEST = APP_ENV=test op run --env-file=".env" --
endif

# Iteration count for `make bench`. Memory benchmarks measure a peak rather than
# an average, so one iteration of a heavy fixture is the useful default.
#
# [Ja] `make bench` の実行回数。メモリのベンチマークは平均ではなくピークを測るため、
# 重いフィクスチャを 1 回実行するのが既定として有用。
BENCHTIME ?= 1x

# Extra flags for `make bench`, for one-off runs such as capturing a profile
# (BENCHFLAGS="-memprofile=tmp/mem.out").
#
# [Ja] `make bench` へ渡す追加フラグ。プロファイルの取得
# (BENCHFLAGS="-memprofile=tmp/mem.out") のような一時的な実行のため。
BENCHFLAGS ?=

.PHONY: help
help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Install the Go module and frontend dependencies. [Ja] Go モジュールとフロントエンドの依存関係をインストール
	go mod download
	pnpm install

.PHONY: run
run: ## サーバーを起動
	$(OP_RUN_DEV) go run ./cmd/mewst serve

.PHONY: seed
seed: ## Rebuild the development database from the seed data. [Ja] 開発用データベースをシードデータで作り直す
	$(OP_RUN_DEV) go run ./cmd/mewst seed

.PHONY: browse-login
browse-login: ## Log in to the dev site in a browser (other user: make browse-login BROWSE_USER=2). [Ja] dev サイトへブラウザでログイン (別ユーザーは make browse-login BROWSE_USER=2)
	$(OP_RUN_DEV) bash scripts/browse.sh login $(BROWSE_USER)

.PHONY: browse-shot
browse-shot: ## Open a page with the logged-in session and screenshot it (e.g. make browse-shot URL=/). [Ja] ログイン済みセッションでページを開きスクショ (例: make browse-shot URL=/)
	bash scripts/browse.sh shot "$(URL)"

.PHONY: browse-close
browse-close: ## Close the browser session and remove temp files. [Ja] ブラウザセッションを閉じ一時ファイルを削除
	bash scripts/browse.sh close

.PHONY: templ-generate
templ-generate: ## templファイルからGoコードを生成し、import文を整理
	@which templ > /dev/null || (echo "Installing templ from go.mod..." && go install github.com/a-h/templ/cmd/templ)
	templ generate
	@$(MAKE) templ-goimports

.PHONY: templ-goimports
templ-goimports: ## Tidy imports for templ-generated files (scope shared with prod Dockerfile) [Ja] templ 生成ファイルの import 文を整理 (本番 Dockerfile と共有するスコープ)
	@which goimports > /dev/null || (echo "Installing goimports from go.mod..." && go install golang.org/x/tools/cmd/goimports)
	goimports -w internal/templates

.PHONY: goimports
goimports: ## goimports で import 文を整理 (リポジトリ全体)
	@which goimports > /dev/null || (echo "Installing goimports from go.mod..." && go install golang.org/x/tools/cmd/goimports)
	goimports -w .

.PHONY: goimports-check
goimports-check: ## goimportsでimport文をチェック（差分があればエラー）
	@which goimports > /dev/null || (echo "Installing goimports from go.mod..." && go install golang.org/x/tools/cmd/goimports)
	@unformatted=$$(goimports -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "The following files have incorrect imports:"; \
		echo "$$unformatted"; \
		echo ""; \
		echo "Please run 'make goimports' to fix imports"; \
		exit 1; \
	fi

.PHONY: build
build: templ-generate ## バイナリをビルド (templ-generate が templ-goimports も実行)
	go build -o bin/mewst ./cmd/mewst

.PHONY: test
test: db-setup-test ## テストを実行
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./... | tparse -all'

.PHONY: test-pkg
test-pkg: db-setup-test ## 特定のパッケージのテストを実行（使用例: make test-pkg PKG=internal/handler/sign_in）
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make test-pkg PKG=internal/handler/sign_in"; \
		exit 1; \
	fi
	@echo "パッケージ $(PKG) のテストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./$(PKG) | tparse -all'

.PHONY: test-run
test-run: db-setup-test ## 特定のテストを実行（使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate）
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate"; \
		exit 1; \
	fi
	@if [ -z "$(RUN)" ]; then \
		echo "エラー: RUN変数を指定してください"; \
		echo "使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate"; \
		exit 1; \
	fi
	@echo "パッケージ $(PKG) のテスト $(RUN) を実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./$(PKG) -run $(RUN) | tparse -all'

.PHONY: test-verbose
test-verbose: db-setup-test ## 詳細ログ付きで全テストを実行（デバッグ用）
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@echo "ローカル環境: 詳細ログ付きで全テストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race -count=1 ./... | tparse -all'

.PHONY: bench
bench: db-setup-test ## Run benchmarks (e.g. make bench PKG=internal/exportfile RUN=BenchmarkBuilder_StreamsArchive). [Ja] ベンチマークを実行 (例: make bench PKG=internal/exportfile RUN=BenchmarkBuilder_StreamsArchive)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make bench PKG=internal/exportfile RUN=BenchmarkBuilder_StreamsArchive"; \
		exit 1; \
	fi
	@echo "パッケージ $(PKG) のベンチマーク $(or $(RUN),.) を実行 (BENCHTIME=$(BENCHTIME))"
	@bash -c '$(OP_RUN_TEST) go test -run "^$$" -bench "$(or $(RUN),.)" -benchmem -benchtime $(BENCHTIME) $(BENCHFLAGS) ./$(PKG)'

.PHONY: test-js
test-js: ## Run frontend unit tests (Vitest). [Ja] フロントエンドのユニットテストを実行 (Vitest)
	pnpm test

.PHONY: typecheck-js
typecheck-js: ## Type-check frontend TypeScript (tsc --noEmit). [Ja] フロントエンドの TypeScript を型検査 (tsc --noEmit)
	pnpm typecheck

.PHONY: fmt
fmt: ## コードをフォーマット
	go fmt ./...
	@$(MAKE) goimports

.PHONY: lint-golangci
lint-golangci: ## golangci-lintを実行
	@echo "🔍 golangci-lintを実行中..."
	@golangci-lint run --config=.golangci.yml ./...

# Base git ref for the diff-scoped comment check. Mirrors CI's PR step, which
# runs against the PR base branch. Override when your PR targets another branch:
# make lint-comments KORYLUSLINT_BASE=origin/main
#
# [Ja] コメント差分チェックの基準 git ref。CI の PR ステップ (PR のベースブランチに対して
# 実行) を再現する。別のブランチを対象にする PR では上書きする。
KORYLUSLINT_BASE ?= origin/develop

.PHONY: lint-comments
lint-comments: ## Check [Ja]-marker misuse in code comments (koryluslint). [Ja] コードコメントの [Ja] マーカー誤用をチェック (koryluslint)
	@echo "🔍 コメントの英日併記をチェック中..."
	@go tool koryluslint comment
	@go tool koryluslint comment -base=$(KORYLUSLINT_BASE)

.PHONY: lint
lint: lint-golangci lint-comments sqlc-diff ## Run the lint checks (golangci-lint + bilingual comments + sqlc sync). [Ja] リントを実行 (golangci-lint + コメント併記チェック + sqlc の同期チェック)

.PHONY: clean
clean: ## ビルド成果物をクリーン
	rm -rf bin/
	find . -name "*_templ.go" -type f -delete

# データベースマイグレーションコマンド

# Strip the random \restrict / \unrestrict lines that pg_dump >= 17.6 emits into
# db/schema.sql, which dbmate writes on both up and dump. Shared by db-migrate
# and db-dump so the workaround lives in one place.
# https://github.com/amacneil/dbmate/issues/678
#
# [Ja] dbmate が up / dump の双方で db/schema.sql へ書き込む、pg_dump 17.6 以降が
# 出力するランダムな \restrict / \unrestrict 行を削除する。db-migrate と db-dump で
# 共有し、回避策を 1 箇所にまとめる。
STRIP_SCHEMA_RESTRICT = sed -i '/^\\restrict /d;/^\\unrestrict /d' db/schema.sql

.PHONY: db-migrate
db-migrate: ## データベースマイグレーションを実行
	$(OP_RUN_DEV) dbmate up
	@$(STRIP_SCHEMA_RESTRICT)

# Initialize the development database without discarding data. `dbmate create`
# succeeds only when the database did not exist, so the schema is loaded exactly
# once, on a fresh environment, and every later run only applies the migrations
# that are still pending. Use db-setup-dev instead to rebuild an existing
# database.
#
# If the initial schema load fails, the database created by that invocation is
# dropped before the error is returned. This lets the same command start cleanly
# on retry. The cleanup runs only after `dbmate create` succeeds, so it can never
# remove a database that existed before the command started.
#
# --no-dump-schema keeps setup from writing db/schema.sql back: dbmate rewrites
# the pg_dump version header on every dump, so without the flag a routine setup
# leaves a tracked file modified. db-migrate remains the command that updates
# the schema file.
#
# [Ja] 開発用データベースを、既存データを捨てずに初期化する。`dbmate create` は
# データベースが存在しなかったときだけ成功するため、スキーマのロードは新規環境の
# 初回だけ行われ、以降の実行は未適用のマイグレーションを進めるだけになる。既存の
# データベースを作り直したいときは db-setup-dev を使う。
#
# 初回のスキーマロードに失敗した場合は、その実行で作成したデータベースを削除して
# からエラーを返す。これにより、同じコマンドを再実行すれば最初から構築し直せる。
# 後片付けは `dbmate create` が成功したときだけ行うため、実行前から存在していた
# データベースを削除することはない。
#
# --no-dump-schema を付けるのはセットアップが db/schema.sql を書き戻さないように
# するため。dbmate はダンプのたびに pg_dump のバージョン見出しを書き換えるため、
# 付けないと通常のセットアップが Git 管理下のファイルを変更した状態にしてしまう。
# スキーマファイルを更新するコマンドは db-migrate のままとする。
.PHONY: db-prepare-dev
db-prepare-dev: ## Initialize the development database if it does not exist, then apply pending migrations. [Ja] 開発用データベースが無ければ構築し、未適用のマイグレーションを適用する
	@if $(OP_RUN_DEV) dbmate create > /dev/null 2>&1; then \
		if ! $(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'; then \
			$(OP_RUN_DEV) dbmate drop; \
			exit 1; \
		fi; \
	fi
	$(OP_RUN_DEV) dbmate --no-dump-schema up

# Rebuild a database from db/schema.sql. The Rails-era tables (users, profiles,
# posts, ...) were created by Rails migrations and have no counterpart under
# db/migrations, so `dbmate up` alone cannot bootstrap an empty database. Only
# db/schema.sql carries the full schema together with every applied version,
# which also leaves `dbmate up` with nothing pending.
#
# The schema is reset with DROP SCHEMA rather than `dbmate drop`, which fails
# whenever anything else holds a connection to the database (a GUI client left
# open is enough). IF EXISTS keeps the reset re-runnable after a run that was
# interrupted between the drop and the create, and client_min_messages hides the
# NOTICE that CASCADE emits for every object it removes.
#
# The schema is applied with psql rather than `dbmate load` so that the same
# steps work where dbmate is not installed, such as the CI runners.
#
# AUTHORIZATION pg_database_owner reproduces the ownership initdb gives the
# public schema. Recreating it without that clause leaves it owned by the
# connecting role, which pg_dump then reports as a schema worth emitting, and
# db/schema.sql picks up a spurious diff on the next dump.
#
# [Ja] データベースを db/schema.sql から作り直す。users / profiles / posts などの
# Rails 期のテーブルは Rails のマイグレーションが作ったもので db/migrations に
# 対応するファイルが無いため、空のデータベースに `dbmate up` しても再現できない。
# 全テーブルと適用済みバージョンの双方を持つのは db/schema.sql だけであり、
# これを流せば `dbmate up` は未適用ゼロになる。
#
# スキーマのリセットに `dbmate drop` ではなく DROP SCHEMA を使うのは、他から
# 接続が張られていると `dbmate drop` が失敗するため (GUI クライアントを開いた
# ままにしているだけで起きる)。IF EXISTS を付けるのは drop と create の間で
# 中断した場合でも再実行で復帰できるようにするためで、client_min_messages は
# CASCADE が削除対象ごとに出す NOTICE を抑制する。
#
# スキーマの適用に `dbmate load` ではなく psql を使うのは、CI ランナーのように
# dbmate が入っていない環境でも同じ手順を使えるようにするため。
#
# AUTHORIZATION pg_database_owner は initdb が public スキーマに与える所有者を
# 再現するためのもの。これを付けずに作り直すと接続ロールが所有者になり、
# pg_dump が public スキーマを出力対象とみなすため、次回のダンプで
# db/schema.sql に不要な差分が入る。
DB_RESET_SCHEMA_SQL = SET client_min_messages TO warning; DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public AUTHORIZATION pg_database_owner;

# db-setup-dev and db-setup-test run the same three steps and differ only in the
# environment their DATABASE_URL is resolved for.
#
# The `dbmate create` error is ignored because it fails once the database
# exists, and because dbmate is absent on the CI runners, where the database is
# created by the postgres service instead. A real failure is reported by the
# reset that follows.
#
# [Ja] db-setup-dev と db-setup-test は同じ 3 ステップを実行し、DATABASE_URL を
# どの環境向けに解決するかだけが異なる。
#
# `dbmate create` のエラーを握り潰すのは、データベースが既にあると失敗するため、
# および CI ランナーには dbmate が入っておらず、データベースは postgres サービスが
# 作るため。実際の失敗は後続のリセットで報告する。
.PHONY: db-setup-dev
db-setup-dev: ## Recreate the development database from db/schema.sql (drops existing data). [Ja] 開発用データベースを db/schema.sql から作り直す (既存データは失われる)
	@$(OP_RUN_DEV) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-setup-test
db-setup-test: ## Recreate the test database from db/schema.sql (drops existing data). [Ja] テスト用データベースを db/schema.sql から作り直す (既存データは失われる)
	@$(OP_RUN_TEST) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-rollback
db-rollback: ## 最後のマイグレーションをロールバック
	$(OP_RUN_DEV) dbmate down

.PHONY: db-new
db-new: ## 新しいマイグレーションファイルを作成（使用法: make db-new name=create_users）
	dbmate new $(name)

.PHONY: db-dump
db-dump: ## データベーススキーマをdb/schema.sqlにダンプ
	$(OP_RUN_DEV) dbmate dump
	@$(STRIP_SCHEMA_RESTRICT)

.PHONY: sqlc-generate
sqlc-generate: ## Generate Go code from SQL queries. [Ja] SQLクエリからGoコードを生成
	go tool sqlc generate

.PHONY: sqlc-diff
sqlc-diff: ## Check that the generated code is in sync with the queries and schema. [Ja] 生成コードがクエリ・スキーマと同期しているかをチェック
	go tool sqlc diff
