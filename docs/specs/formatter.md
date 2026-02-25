# フォーマッタ 仕様書

<!--
このテンプレートの使い方:
1. 操作対象のモデルに対応するディレクトリを `docs/specs/` 配下に作成（例: `docs/specs/page/`）
2. このファイルをそのディレクトリにコピー（例: cp docs/specs/template.md docs/specs/page/create.md）
3. [機能名] などのプレースホルダーを実際の内容に置き換え
4. 各セクションのガイドラインに従って記述
5. コメント（ `\<!-- ... --\>` ）はガイドラインとして残してください

**ファイルの配置ルール**:
- 仕様書は操作対象のモデル（名詞）ごとにディレクトリを分け、機能（動詞）をファイル名にする
  - 例: `docs/specs/user/sign-up.md`、`docs/specs/page/create.md`
- モデルに分類しにくい横断的な機能は、その機能自体を名詞としてディレクトリにする
  - 例: `docs/specs/search/full-text.md`
- モデルの定義・状態遷移・他モデルとの関係を記述する場合は `overview.md` を作成する
  - `overview.md` はモデルの静的な性質（「これは何か」）を書く場所
  - 操作に紐づく仕様（バリデーション、権限など）は各機能の仕様書に書く
- 詳細は [@docs/README.md](/workspace/docs/README.md) を参照

**仕様書の性質**:
- 仕様書は「現在のシステムの状態」を記述するドキュメントです
- 実装が完了したら、仕様書を最新の状態に更新してください
- 過去の状態はGit履歴で参照できるため、仕様書には常に現在の状態のみを記述します

**作業計画書との関係**:
- 新しい機能の場合: `docs/plans/` の作業計画書に概要・要件・設計を記述し、タスク完了後にこの仕様書を作成します
- 既存機能の変更の場合: `docs/plans/` の作業計画書に変更内容を記述し、タスク完了後にこの仕様書を更新します

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 概要

<!--
ガイドライン:
- この機能が現在「どのように動いているか」を簡潔に説明
- なぜこの仕組みになっているかの背景も記述
- 2-3段落程度で簡潔に
-->

プロジェクト全体のコードフォーマッタとして [Oxfmt](https://oxc.rs/docs/guide/usage/formatter) を使用している。プロジェクトルートから `make fmt` / `make fmt-check` を実行することで、Go プロジェクト・Rails プロジェクト・ルートレベルのファイルを横断してフォーマットを適用できる。

**目的**:

- プロジェクト全体で一貫したコードフォーマットを維持する
- JavaScript, TypeScript, CSS, JSON, YAML, Markdown, TOML ファイルを統一的にフォーマットする

**背景**:

- Oxfmt は Rust 製で Prettier の約 30 倍高速に動作する
- Prettier 100% 互換（JS/TS）で、Tailwind CSS クラスソートを内蔵しているため、プラグインなしで利用できる
- Go プロジェクトが既に pnpm を使用しているため、パッケージマネージャーは pnpm に統一している

## 仕様

<!--
ガイドライン:
- 現在のシステムの振る舞いを記述
- 「ユーザーは〇〇できる」「システムは〇〇する」という形式で記述
- 必要に応じて非機能的な仕様（セキュリティ、パフォーマンスなど）も記述
-->

- プロジェクトルートから `make fmt` を実行して、プロジェクト全体の対象ファイルを自動フォーマットできる
- プロジェクトルートから `make fmt-check` を実行して、フォーマット差分の有無をチェックできる
- CI（GitHub Actions）で `pnpm oxfmt:check` が自動実行され、フォーマット違反を検出する
- フォーマット対象: JavaScript, TypeScript, CSS, SCSS, JSON, JSONC, YAML, Markdown, TOML ファイル
- フォーマットルール: printWidth 120, tabWidth 2, ダブルクォート, トレイリングカンマ

## 設計

<!--
ガイドライン:
- 現在の技術的な実装の詳細を記述
- 必要に応じて以下のようなサブセクションを追加してください：
  - 技術スタック（使用するライブラリ、フレームワーク、ツールなど）
  - アーキテクチャ（システム全体の構成、コンポーネント間の関係など）
  - データベース設計（テーブル定義、インデックス、制約など）
  - API設計（エンドポイント、リクエスト/レスポンス形式など）
  - セキュリティ設計（認証・認可、トークン管理、Rate Limitingなど）
  - コード設計（パッケージ構成、主要な構造体、インターフェースなど）
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### 技術スタック

- **Oxfmt**: Rust 製の高速フォーマッタ（`oxfmt` npm パッケージ）
- **pnpm**: パッケージマネージャー（Go プロジェクトと統一）
- **Make**: `make fmt` / `make fmt-check` でフォーマットコマンドを実行

### ファイル構成

```
/workspace/
├── package.json          # oxfmt の依存関係と npm scripts
├── pnpm-lock.yaml        # pnpm のロックファイル
├── Makefile              # make fmt / make fmt-check ターゲット
├── .oxfmtignore          # フォーマット除外設定
└── .github/
    └── workflows/
        └── fmt-ci.yml    # フォーマットチェック CI
```

### npm scripts

`package.json` に以下の scripts を定義している:

- `oxfmt`: `oxfmt --ignore-path .oxfmtignore` — 自動フォーマット
- `oxfmt:check`: `oxfmt --ignore-path .oxfmtignore --check` — フォーマットチェック

### Makefile ターゲット

プロジェクトルートの `Makefile` に以下のターゲットを定義している:

- `make fmt`: `pnpm oxfmt` を実行してフォーマットを適用
- `make fmt-check`: `pnpm oxfmt:check` を実行してフォーマット差分をチェック

Rails ディレクトリ内からは `make -C /workspace fmt` で呼び出せる。

### フォーマット対象

- `*.js`, `*.jsx`, `*.ts`, `*.tsx` — JavaScript / TypeScript
- `*.css`, `*.scss` — スタイルシート
- `*.json`, `*.jsonc` — JSON
- `*.yaml`, `*.yml` — YAML
- `*.md` — Markdown
- `*.toml` — TOML

### フォーマット除外対象（.oxfmtignore）

- `**/node_modules/` — 依存パッケージ
- `**/vendor/` — バンドルされた Gem
- `rails/db/` — DB スキーマ・マイグレーション（自動生成）
- `rails/docs/`, `rails/fixtures/` — ドキュメント・フィクスチャ
- `rails/app/assets/builds/` — ビルド成果物
- `rails/sorbet/` — Sorbet 自動生成ファイル
- `go/internal/query/*.go` — sqlc 自動生成コード
- `go/internal/templates/*_templ.go` — templ 自動生成コード
- `go/static/` — ビルド成果物

### CI ワークフロー

`.github/workflows/fmt-ci.yml` でプロジェクト全体のフォーマットチェックを実行する。`main` ブランチへの push と pull request をトリガーとし、`pnpm oxfmt:check` で差分がないことを検証する。

### ERB テンプレートのフォーマット

ERB テンプレート（`.html.erb`）は Oxfmt のフォーマット対象外としている。ERB テンプレートのフォーマットは `erb_lint` が担当する。

## 採用しなかった方針

<!--
ガイドライン:
- 検討したが採用しなかった設計や機能を、理由とともに記述
- 将来の開発者が同じ検討を繰り返さないための判断記録として活用する
- 後から実装された場合は、該当項目を削除する
- 該当がない場合も、セクション自体は残しておく（後から追加しやすくするため）
-->

### Prettier の継続使用

Oxfmt 導入前は Rails プロジェクトのみに Prettier を導入していた。Go プロジェクトやルートレベルの Markdown ファイルはフォーマット対象外で、プロジェクト全体で一貫したフォーマットが行われていなかった。Oxfmt は Prettier の約 30 倍高速であること、`oxfmt --migrate prettier` で設定を移行できること、Tailwind CSS クラスソートが内蔵されていることから、Oxfmt への移行を決定した。

### dprint の採用

Rust 製の高速フォーマッタ dprint も検討した。プラグイン方式で柔軟性が高いが、Oxfmt は Prettier との互換性が高く移行コマンドが用意されている点、Tailwind CSS クラスソートが内蔵されている点で Oxfmt を選定した。

### Rails ディレクトリのみで Oxfmt を使用する方針

Rails の `package.json` に oxfmt を追加する方針も検討したが、プロジェクト全体で統一的にフォーマットを適用するためルートレベルに配置する方針とした。

### ERB テンプレートの Tailwind クラスソートを Oxfmt で行う方針

Oxfmt は HTML の Tailwind クラスソートをネイティブサポートしているが、ERB テンプレートは標準的な HTML ではなく Ruby コードが混在するため、対応状況が不明。ERB のフォーマットは `erb_lint` が担当しているため、ERB を Oxfmt の対象外とした。

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Oxfmt 公式ドキュメント - Usage](https://oxc.rs/docs/guide/usage/formatter)
- [Oxfmt Beta アナウンス](https://oxc.rs/blog/2026-02-24-oxfmt-beta)
