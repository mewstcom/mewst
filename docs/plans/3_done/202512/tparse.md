# tparse 導入 仕様書

## 概要

Go 版のテスト実行結果の可読性を向上させるため、[tparse](https://github.com/mfridman/tparse) を導入します。

tparse は `go test` の JSON 出力を解析・要約するコマンドラインツールです。テスト失敗とパニックを強調表示し、パッケージレベルのサマリーテーブルを出力することで、テスト結果を素早く把握できるようになります。

**目的**:

- テスト結果の可読性を向上させ、失敗したテストを素早く特定できるようにする
- パッケージレベルのサマリーでテスト全体の状況を把握しやすくする
- CI 環境でも見やすいテスト結果を出力する

**背景**:

- 現在の `go test -v` 出力は冗長で、テスト数が増えると失敗したテストを見つけにくい
- 特に複数パッケージにまたがるテストでは、どのパッケージが失敗したかの把握に時間がかかる

## 要件

### 機能要件

- `make test` コマンドでテストを実行した際、tparse を経由して結果を整形表示する
- `make test-pkg`、`make test-run`、`make test-verbose` コマンドでも同様に tparse を使用する
- 失敗したテストとパニックは強調表示される
- パッケージレベルのサマリーテーブルが表示される
- CI 環境でも正常に動作する

### 非機能要件

- tparse は `tools.go` で管理し、バージョンを固定する
- 既存の Makefile タスクとの互換性を維持する（コマンド名は変更しない）
- `set -o pipefail` を使用してパイプラインのエラーを適切に検出する

## 設計

### 技術スタック

- **tparse v0.18.0**: Go テスト出力解析ツール

### 使用方法

tparse は `go test -json` の出力をパイプで受け取って使用します：

```sh
# 基本的な使い方
set -o pipefail && go test -json ./... | tparse

# 失敗テストのみ表示（デフォルト）
set -o pipefail && go test -json ./... | tparse

# 合格テストも表示
set -o pipefail && go test -json ./... | tparse -all
```

### Makefile の変更

既存の test 関連タスクを tparse を使用するように変更します。

**変更前**:

```makefile
test: db-setup-test
	go test -v -race ./...
```

**変更後**:

```makefile
test: db-setup-test
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse@latest)
	set -o pipefail && go test -json -race ./... | tparse
```

### tools.go への追加

```go
//go:build tools

package tools

import (
	_ "github.com/a-h/templ/cmd/templ"
	_ "github.com/mfridman/tparse"
	_ "golang.org/x/tools/cmd/goimports"
)
```

## タスクリスト

### フェーズ 1: tparse の導入

- [x] **1-1**: tparse を tools.go に追加し、go.mod に依存関係を追加する
  - tools.go に `_ "github.com/mfridman/tparse"` を追加
  - `go get github.com/mfridman/tparse@v0.18.0` を実行
  - `go mod tidy` で依存関係を整理
  - **想定ファイル数**: 約 2 ファイル（実装 2 + テスト 0）
  - **想定行数**: 約 5 行（実装 5 行 + テスト 0 行）

- [x] **1-2**: Makefile の test 関連タスクを tparse を使用するように変更する
  - `make test` を tparse 経由に変更
  - `make test-pkg` を tparse 経由に変更
  - `make test-run` を tparse 経由に変更
  - `make test-verbose` を tparse 経由に変更（-all オプション付き）
  - CI 環境用の分岐も tparse を使用するように変更
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 30 行（実装 30 行 + テスト 0 行）

- [x] **1-3**: CLAUDE.md のドキュメントを更新する
  - テスト実行コマンドの説明に tparse について追記
  - **想定ファイル数**: 約 1 ファイル（実装 1 + テスト 0）
  - **想定行数**: 約 10 行（実装 10 行 + テスト 0 行）

### 実装しない機能（スコープ外）

以下の機能は今回の実装では**実装しません**：

- **カバレッジレポートの統合**: tparse はカバレッジ表示もサポートしているが、今回はテスト結果の整形のみに焦点を当てる
- **GitHub Actions との統合**: tparse の GitHub Actions 向け機能（`-format markdown` など）は将来の検討事項とする

## 参考資料

- [tparse GitHub リポジトリ](https://github.com/mfridman/tparse)
- [tparse リリースノート v0.18.0](https://github.com/mfridman/tparse/releases/tag/v0.18.0)
