# op runラッパー: 1Password CLI経由で .envの環境変数を解決してコマンドを実行する
# CI環境ではopコマンドが利用できないため、環境変数はGitHub Actionsで直接設定する
OP_RUN_DEV = APP_ENV=dev op run --env-file=".env" --
ifeq ($(CI),true)
OP_RUN_TEST = APP_ENV=test
else
OP_RUN_TEST = APP_ENV=test op run --env-file=".env" --
endif

# `make bench` の実行回数。メモリのベンチマークは平均ではなくピークを測るため、
# 重いフィクスチャを1回実行するのが既定として有用。
BENCHTIME ?= 1x

# `make bench` へ渡す追加フラグ。プロファイルの取得
# (BENCHFLAGS="-memprofile=tmp/mem.out") のような一時的な実行のため。
BENCHFLAGS ?=

.PHONY: help
help: ## ヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Goモジュールとフロントエンドの依存関係をインストール
	go mod download
	pnpm install

.PHONY: run
run: ## サーバーを起動
	$(OP_RUN_DEV) go run ./cmd/mewst serve

.PHONY: seed
seed: ## 開発用データベースをシードデータで作り直す
	$(OP_RUN_DEV) go run ./cmd/mewst seed

.PHONY: browse-login
browse-login: ## seed-users.tomlの役割でdevサイトへブラウザでログイン (別の役割はmake browse-login BROWSE_USER=follower)
	$(OP_RUN_DEV) bash scripts/browse.sh login "$(BROWSE_USER)"

.PHONY: browse-shot
browse-shot: ## ログイン済みセッションでページを開きスクショ (例: make browse-shot URL=/)
	bash scripts/browse.sh shot "$(URL)"

.PHONY: browse-close
browse-close: ## ブラウザセッションを閉じ一時ファイルを削除
	bash scripts/browse.sh close

.PHONY: templ-generate
templ-generate: ## templファイルからGoコードを生成し、import文を整理
	@which templ > /dev/null || (echo "Installing templ from go.mod..." && go install github.com/a-h/templ/cmd/templ)
	templ generate
	@$(MAKE) templ-goimports

.PHONY: templ-goimports
templ-goimports: ## templ生成ファイルのimport文を整理 (本番Dockerfileと共有するスコープ)
	@which goimports > /dev/null || (echo "Installing goimports from go.mod..." && go install golang.org/x/tools/cmd/goimports)
	goimports -w internal/templates

.PHONY: goimports
goimports: ## goimportsでimport文を整理 (リポジトリ全体)
	@which goimports > /dev/null || (echo "Installing goimports from go.mod..." && go install golang.org/x/tools/cmd/goimports)
	goimports -w .

.PHONY: goimports-check
goimports-check: ## goimportsでimport文をチェック (差分があればエラー)
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
build: templ-generate ## バイナリをビルド (templ-generateがtempl-goimportsも実行)
	go build -o bin/mewst ./cmd/mewst

.PHONY: test
test: db-setup-test ## テストを実行
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./... | tparse -all'

.PHONY: test-pkg
test-pkg: db-setup-test ## 特定のパッケージのテストを実行 (使用例: make test-pkg PKG=internal/handler/sign_in)
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make test-pkg PKG=internal/handler/sign_in"; \
		exit 1; \
	fi
	@echo "パッケージ $(PKG) のテストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race ./$(PKG) | tparse -all'

.PHONY: test-run
test-run: db-setup-test ## 特定のテストを実行 (使用例: make test-run PKG=internal/handler/sign_in RUN=TestCreate)
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
test-verbose: db-setup-test ## 詳細ログ付きで全テストを実行 (デバッグ用)
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@echo "ローカル環境: 詳細ログ付きで全テストを実行"
	@bash -c 'set -o pipefail && $(OP_RUN_TEST) go test -json -race -count=1 ./... | tparse -all'

.PHONY: bench
bench: db-setup-test ## ベンチマークを実行 (例: make bench PKG=internal/exportfile RUN=BenchmarkBuilder_StreamsArchive)
	@if [ -z "$(PKG)" ]; then \
		echo "エラー: PKG変数を指定してください"; \
		echo "使用例: make bench PKG=internal/exportfile RUN=BenchmarkBuilder_StreamsArchive"; \
		exit 1; \
	fi
	@echo "パッケージ $(PKG) のベンチマーク $(or $(RUN),.) を実行 (BENCHTIME=$(BENCHTIME))"
	@bash -c '$(OP_RUN_TEST) go test -run "^$$" -bench "$(or $(RUN),.)" -benchmem -benchtime $(BENCHTIME) $(BENCHFLAGS) ./$(PKG)'

.PHONY: test-js
test-js: ## フロントエンドのユニットテストを実行 (Vitest)
	pnpm test

.PHONY: typecheck-js
typecheck-js: ## フロントエンドのTypeScriptを型検査 (tsc --noEmit)
	pnpm typecheck

.PHONY: fmt
fmt: ## コードをフォーマット
	go fmt ./...
	@$(MAKE) goimports

.PHONY: lint-golangci
lint-golangci: ## golangci-lintを実行
	@echo "🔍 golangci-lintを実行中..."
	@golangci-lint run --config=.golangci.yml ./...

.PHONY: lint-style
lint-style: ## 日本語テキストの表記をチェック (koryluslint)
	@go tool koryluslint comment

.PHONY: lint
lint: lint-golangci lint-style sqlc-diff ## リントを実行 (golangci-lint + 日本語テキストの表記 + sqlcの同期チェック)

.PHONY: clean
clean: ## ビルド成果物をクリーン
	rm -rf bin/
	find . -name "*_templ.go" -type f -delete

# データベースマイグレーションコマンド

# dbmateがup / dumpの双方でdb/schema.sqlへ書き込む、pg_dump 17.6以降が
# 出力するランダムな \restrict / \unrestrict行を削除する。db-migrateとdb-dumpで
# 共有し、回避策を1箇所にまとめる。
# https://github.com/amacneil/dbmate/issues/678
STRIP_SCHEMA_RESTRICT = sed -i '/^\\restrict /d;/^\\unrestrict /d' db/schema.sql

.PHONY: db-migrate
db-migrate: ## データベースマイグレーションを実行
	$(OP_RUN_DEV) dbmate up
	@$(STRIP_SCHEMA_RESTRICT)

# 開発用データベースを、既存データを捨てずに初期化する。`dbmate create` は
# データベースが存在しなかったときだけ成功するため、スキーマのロードは新規環境の
# 初回だけ行われ、以降の実行は未適用のマイグレーションを進めるだけになる。既存の
# データベースを作り直したいときはdb-setup-devを使う。
#
# 初回のスキーマロードに失敗した場合は、その実行で作成したデータベースを削除して
# からエラーを返す。これにより、同じコマンドを再実行すれば最初から構築し直せる。
# 後片付けは `dbmate create` が成功したときだけ行うため、実行前から存在していた
# データベースを削除することはない。
#
# --no-dump-schemaを付けるのはセットアップがdb/schema.sqlを書き戻さないように
# するため。dbmateはダンプのたびにpg_dumpのバージョン見出しを書き換えるため、
# 付けないと通常のセットアップがGit管理下のファイルを変更した状態にしてしまう。
# スキーマファイルを更新するコマンドはdb-migrateのままとする。
.PHONY: db-prepare-dev
db-prepare-dev: ## 開発用データベースが無ければ構築し、未適用のマイグレーションを適用する
	@if $(OP_RUN_DEV) dbmate create > /dev/null 2>&1; then \
		if ! $(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'; then \
			$(OP_RUN_DEV) dbmate drop; \
			exit 1; \
		fi; \
	fi
	$(OP_RUN_DEV) dbmate --no-dump-schema up

# データベースをdb/schema.sqlから作り直す。users / profiles / postsなどの
# Rails期のテーブルはRailsのマイグレーションが作ったものでdb/migrationsに
# 対応するファイルが無いため、空のデータベースに `dbmate up` しても再現できない。
# 全テーブルと適用済みバージョンの双方を持つのはdb/schema.sqlだけであり、
# これを流せば `dbmate up` は未適用ゼロになる。
#
# スキーマのリセットに `dbmate drop` ではなくDROP SCHEMAを使うのは、他から
# 接続が張られていると `dbmate drop` が失敗するため (GUIクライアントを開いた
# ままにしているだけで起きる)。IF EXISTSを付けるのはdropとcreateの間で
# 中断した場合でも再実行で復帰できるようにするためで、client_min_messagesは
# CASCADEが削除対象ごとに出すNOTICEを抑制する。
#
# スキーマの適用に `dbmate load` ではなくpsqlを使うのは、CIランナーのように
# dbmateが入っていない環境でも同じ手順を使えるようにするため。
#
# AUTHORIZATION pg_database_ownerはinitdbがpublicスキーマに与える所有者を
# 再現するためのもの。これを付けずに作り直すと接続ロールが所有者になり、
# pg_dumpがpublicスキーマを出力対象とみなすため、次回のダンプで
# db/schema.sqlに不要な差分が入る。
DB_RESET_SCHEMA_SQL = SET client_min_messages TO warning; DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public AUTHORIZATION pg_database_owner;

# db-setup-devとdb-setup-testは同じ3ステップを実行し、DATABASE_URLを
# どの環境向けに解決するかだけが異なる。
#
# `dbmate create` のエラーを握り潰すのは、データベースが既にあると失敗するため、
# およびCIランナーにはdbmateが入っておらず、データベースはpostgresサービスが
# 作るため。実際の失敗は後続のリセットで報告する。
.PHONY: db-setup-dev
db-setup-dev: ## 開発用データベースをdb/schema.sqlから作り直す (既存データは失われる)
	@$(OP_RUN_DEV) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_DEV) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-setup-test
db-setup-test: ## テスト用データベースをdb/schema.sqlから作り直す (既存データは失われる)
	@$(OP_RUN_TEST) sh -c 'dbmate create > /dev/null 2>&1 || true'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -c "$(DB_RESET_SCHEMA_SQL)" > /dev/null'
	@$(OP_RUN_TEST) sh -c 'psql "$$DATABASE_URL" -v ON_ERROR_STOP=1 -q -f db/schema.sql > /dev/null'

.PHONY: db-rollback
db-rollback: ## 最後のマイグレーションをロールバック
	$(OP_RUN_DEV) dbmate down

.PHONY: db-new
db-new: ## 新しいマイグレーションファイルを作成 (使用法: make db-new name=create_users)
	dbmate new $(name)

.PHONY: db-dump
db-dump: ## データベーススキーマをdb/schema.sqlにダンプ
	$(OP_RUN_DEV) dbmate dump
	@$(STRIP_SCHEMA_RESTRICT)

.PHONY: sqlc-generate
sqlc-generate: ## SQLクエリからGoコードを生成
	go tool sqlc generate

.PHONY: sqlc-diff
sqlc-diff: ## 生成コードがクエリ・スキーマと同期しているかをチェック
	go tool sqlc diff
