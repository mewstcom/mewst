.PHONY: test test-file setup console server worker lint lint-fix fmt zeitwerk db-setup-test help

# Set the default value for APP_ENV (a value given from the outside wins).
# e.g. APP_ENV=test make console
#
# [Ja] APP_ENV のデフォルト値を設定 (外部から指定された場合はその値を使用)
# 例: APP_ENV=test make console
APP_ENV ?= dev

# 1Password CLI経由で環境変数を読み込んでコマンドを実行
OP_RUN := APP_ENV=$(APP_ENV) op run --env-file=".env.local" --
# テスト専用（常にtest環境）
OP_RUN_TEST := APP_ENV=test op run --env-file=".env.local" --

## テスト
test: db-setup-test ## 全テストを実行
	$(OP_RUN_TEST) bin/rspec

test-file: db-setup-test ## 特定のテストファイルを実行 (例: make test-file FILE=spec/models/user_spec.rb)
	$(OP_RUN_TEST) bin/rspec $(FILE)

# bin/setup does not connect to the database, but the log:clear / tmp:clear it
# runs go through Rake, which resolves config/database.yml. The op run wrapper
# supplies the environment variables that resolution needs.
#
# [Ja] bin/setup はデータベースへ接続しないが、その中で実行する log:clear /
# tmp:clear が Rake 経由で config/database.yml を解決する。op run のラッパーは
# その解決に必要な環境変数を与える。
setup: ## セットアップコマンド
	$(OP_RUN) bin/setup

## 開発サーバー
console: ## Railsコンソールを起動
	$(OP_RUN) bin/rails console

server: ## Railsサーバーを起動
	$(OP_RUN) bin/rails server

worker: ## Railsワーカーを起動
	$(OP_RUN) bin/good_job start

## リント・フォーマット
lint: ## Rubyコードのリントを実行
	$(OP_RUN) bin/standardrb

lint-fix: ## Rubyコードのリントを実行して自動修正
	$(OP_RUN) bin/standardrb --fix

fmt: lint-fix ## lint-fixのエイリアス

## オートロードチェック
zeitwerk: ## Zeitwerk（オートロード）チェック
	$(OP_RUN) bin/rails zeitwerk:check

## データベース
# Rebuild the test database before running specs, by delegating to the Go side
# that owns it. The schema comes from go/db/schema.sql, and schema_dump: false
# in config/database.yml stops ActiveRecord::Migration.maintain_test_schema!
# from refreshing it, so without this nothing notices when the Go schema moves
# ahead of the local database and the specs keep running against a stale one.
#
# [Ja] spec の実行前に、テスト用データベースをそれを所有する Go 側へ委譲して
# 作り直す。スキーマは go/db/schema.sql が正本であり、config/database.yml の
# schema_dump: false によって ActiveRecord::Migration.maintain_test_schema! も
# 更新しなくなったため、これが無いと Go 側のスキーマが進んでもローカルの
# データベースが古いままであることに気づけず、古いスキーマのまま spec が走る。
db-setup-test: ## Rebuild the test database from the Go schema. [Ja] Go 側のスキーマからテスト用データベースを作り直す
	$(MAKE) -C ../go db-setup-test

## ヘルプ
help: ## このヘルプを表示
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
