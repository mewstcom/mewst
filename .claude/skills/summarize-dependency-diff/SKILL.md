---
name: summarize-dependency-diff
description: 依存パッケージの old → new バージョン間の実差分（CHANGELOG / CVE / Breaking change）を取得し、指定された一時ドキュメントに「バージョン間差分サマリー」セクションとして追記する。`/review-pr` および `/fix-dependabot-alert` から呼び出される sub-skill。
argument-hint: "<ecosystem> <package> <old_version> <new_version> <output_path>"
---

# 依存パッケージの差分サマリー作成 <ecosystem> <package> <old_version> <new_version> <output_path>

依存パッケージの 2 バージョン間の実差分を取得し、CHANGELOG / Releases / コミットログから CVE 修正・Breaking change・挙動変更・推移的依存の変化を抽出して、指定された一時ドキュメントに `## バージョン間差分サマリー` セクションを追記してください。

このスキルは `/review-pr` と `/fix-dependabot-alert` の両方から呼び出される共通処理です。SemVer の表面的な情報だけで「パッチだから安全」と判断しないために、依存ライブラリの実差分を必ず確認するためのものです。

> パッチアップデートでも behavior change や Breaking change を取り込むライブラリは多い（Rails、Sentry、grpc、AWS SDK など）。

## 引数

`$ARGUMENTS` は以下の形式で渡されます:

- 形式: `<ecosystem> <package> <old_version> <new_version> <output_path>`
- 例: `rubygems erb 5.0.3 6.0.4 tmp/fix-dependabot-alert/alert-170.md`
- 例: `npm flatted 3.4.1 3.4.2 tmp/review-pr/pr-152.md`

引数の意味:

| 引数          | 内容                                                           |
| ------------- | -------------------------------------------------------------- |
| `ecosystem`   | `rubygems` / `npm` / `gomod` / `pip` / `composer` のいずれか   |
| `package`     | パッケージ名（npm scope 付き `@scope/pkg` も含む）             |
| `old_version` | 旧バージョン（例: `5.0.3`）                                    |
| `new_version` | 新バージョン（例: `6.0.4`）                                    |
| `output_path` | 追記対象の Markdown ファイルパス（呼び出し側が事前に作成済み） |

引数の数が足りない・形式が不正な場合はエラーを報告して終了してください。

## 原則

- **追記する**: `output_path` のファイルは呼び出し側で作成済み。新規作成や上書きはしない。`## バージョン間差分サマリー` セクションをファイル末尾に追記する
- **失敗してもスキル全体は止めない**: ソースリポジトリ特定不能・タグ命名不明・GitHub 以外のホスティングなどのコーナーケースでは、「差分取得不能」と明記したサマリーを追記してスキルを正常終了する。`/review` 側に「コードベースで広めに grep してほしい」と促す
- **best-effort で fallback する**: CHANGELOG → GitHub Releases → タグ間 commit log の順に試行する
- **巻き込み更新の検出を忘れない**: 推移的依存の追加・削除・メジャーアップは Breaking change と並んで重要。`Gemfile.lock` 差分などからも掴めるが、ここではアップストリームの CHANGELOG / Releases に記載されたものを記録する

## 手順

### ステップ 1: 引数の解析

- `$ARGUMENTS` をスペースで分割し、5 個の位置引数を取り出す
- 5 個に満たない場合はエラーを報告して終了する
- `output_path` のファイルが存在することを `ls` 等で確認する。存在しない場合はエラーを報告して終了する

### ステップ 2: ソースリポジトリの特定

ecosystem ごとに、パッケージのソースリポジトリ URL を取得する。GitHub 上のリポジトリのみを対象とし、それ以外（GitLab・Bitbucket・自前ホスティングなど）は「差分取得不能」として扱う。

| ecosystem  | 取得方法                                                                                                                                  |
| ---------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `rubygems` | `curl -s https://rubygems.org/api/v1/gems/<package>.json \| jq -r '.source_code_uri // .homepage_uri'`                                    |
| `npm`      | `curl -s https://registry.npmjs.org/<package> \| jq -r '.repository.url // .homepage'`（`git+https://...` プレフィックスを除去する）      |
| `gomod`    | パッケージパスがそのままリポジトリ（例: `github.com/foo/bar/v2` → `github.com/foo/bar`）。`golang.org/x/...` などは GitHub にミラーされる |
| `pip`      | `curl -s https://pypi.org/pypi/<package>/json \| jq -r '.info.project_urls.Source // .info.home_page'`                                    |
| `composer` | `curl -s https://repo.packagist.org/p2/<package>.json \| jq -r '.packages."<package>"[0].source.url'`                                     |

取得した URL を正規化して `<owner>/<repo>` 形式にする。GitHub 以外のホストの場合は **ステップ 7（差分取得不能）** に進む。

### ステップ 3: タグ命名規則の特定

リポジトリのタグ一覧を取得し、`old_version` / `new_version` に対応するタグ名を推定する。

```sh
gh api repos/<owner>/<repo>/tags --jq '.[].name' | head -100
```

代表的なパターン:

- `v1.2.3`（Ruby gem 系・npm 系で多い）
- `1.2.3`（Go modules・一部の Ruby gem）
- `<package>-v1.2.3` / `<package>/v1.2.3`（モノレポ）
- `release-1.2.3` / `releases/1.2.3`（一部のプロジェクト）

タグ一覧の中から `old_version` / `new_version` 文字列を含むタグをそれぞれ 1 つずつ選ぶ。両方が見つからない場合はページングして追加で取得する（`--paginate` オプション）。

```sh
gh api --paginate repos/<owner>/<repo>/tags --jq '.[].name' | grep -E "(^|[^0-9])<old_version>([^0-9]|$)"
```

両方見つからない場合は **ステップ 7（差分取得不能）** に進む。

### ステップ 4: 実差分の規模把握

```sh
gh api repos/<owner>/<repo>/compare/<old_tag>...<new_tag> \
  --jq '{total_commits: .total_commits, files_count: (.files | length)}'
```

- 総コミット数とファイル数を取得し、規模感を把握する
- パッチアップデートでも数百コミット含まれることがある（マイナーリリースを跨ぐ場合）
- compare API がエラー（例: tag が直接比較不可、同一の SHA など）を返した場合は **ステップ 7** に進む

### ステップ 5: CHANGELOG / CHANGES / HISTORY の差分取得

```sh
gh api repos/<owner>/<repo>/compare/<old_tag>...<new_tag> \
  --jq '.files[] | select(.filename | test("(?i)CHANGELOG\\.md$|CHANGES\\.md$|HISTORY\\.md$|NEWS\\.md$|CHANGELOG$|CHANGES$")) | "=== \(.filename) ===\n\(.patch)\n"'
```

出力が大きい場合（数千行を超える）は `head` などで分割して取得する。複数の CHANGELOG ファイルがある（モノレポなど）場合はそれぞれを抽出する。

### ステップ 6: CHANGELOG が空の場合の fallback

CHANGELOG が見つからなかった、または `.patch` が空（バイナリ的な変更や巨大すぎて GitHub API が patch を返さなかった）の場合は、以下の順で fallback する。

#### 6-A: GitHub Releases

```sh
gh api repos/<owner>/<repo>/releases --jq '.[] | select(.tag_name >= "<old_tag>" and .tag_name <= "<new_tag>") | {tag: .tag_name, name: .name, body: .body}'
```

`old_tag` < tag <= `new_tag` の範囲（old は除外、new は含める）の Release ノートを集める。

#### 6-B: タグ間の commit log 要約

Releases も無い場合は commit log を集約する。

```sh
gh api repos/<owner>/<repo>/compare/<old_tag>...<new_tag> --jq '.commits[] | "- \(.commit.message | split("\n")[0])"'
```

主要な変更（fix / feat / break / security 等のキーワードを含むコミット）を要約する。

すべてが空だった場合は **ステップ 7** に進む。

### ステップ 7: 差分取得不能で終了する場合の処理

以下のいずれかに該当する場合、差分取得不能として扱う:

- ソースリポジトリが GitHub 以外
- ソースリポジトリ URL が取得できない
- old / new に対応するタグが特定できない
- compare API がエラーを返した
- CHANGELOG / Releases / commit log のいずれからも有用な情報が取れなかった

この場合、`output_path` に「差分取得不能」を明記したサマリーを追記する（テンプレートのうち基本情報と「### 差分取得不能」セクションのみを埋める）。`/review` 側ではこのシグナルを受けて、コードベースで広めに grep して影響範囲を確認することになる。

### ステップ 8: 抽出する情報

ステップ 5 / 6 で得た本文から、以下を抽出する:

- **追加されている CVE 修正**: `CVE-` を含む行・「security」「vulnerability」「advisory」を含む行
- **Breaking change と明記された変更**: `BREAKING` / `breaking change` / `removed` / `Note that this change breaks` / `Backwards incompatible` などの文言
- **挙動が変わる修正**: `Fix XX to return ...` のような戻り値変更、SQL 生成パターンの変更、バリデーション条件の変更、デフォルト値変更、エラー型変更など
- **新規依存・依存削除・メジャーアップ**: 「Add dependency」「Drop support」「Bump <dep> from X to Y」など

メジャーアップを跨ぐケース（例: 5.x → 6.x）では、特に Breaking change の見落としを警戒する。

### ステップ 9: 出力先への追記

`output_path` のファイル末尾に以下のセクションを追記する。既に同じセクションが存在する場合は上書きせず、`## バージョン間差分サマリー (再実行)` のように見出しに追記して新しいセクションとして残す（履歴を消さない）。

```markdown
## バージョン間差分サマリー

- **対象パッケージ**: `<package>` (<ecosystem>)
- **バージョン**: <old_version> → <new_version>
- **ソースリポジトリ**: <https://github.com/<owner>/<repo>>
- **Compare view URL**: <https://github.com/<owner>/<repo>/compare/<old_tag>...<new_tag>>
- **実差分の規模**: <N commits / N files>
- **間に含まれるリリース**: <例: 6.0.1, 6.0.1.1, 6.0.2, 6.0.3, 6.0.4>
- **取得元**: <CHANGELOG / GitHub Releases / commit log / 差分取得不能>

### CHANGELOG から抽出した CVE 修正

<!-- 該当がない場合: 「該当なし」と記載 -->

| CVE            | 内容         | 出典バージョン |
| -------------- | ------------ | -------------- |
| CVE-XXXX-XXXXX | <内容の要約> | <version>      |

### CHANGELOG から抽出した Breaking change / 挙動変更

<!-- 該当がない場合: 「該当なし」と記載 -->

| 変更         | 出典バージョン | 影響を受けるパターン（grep キーワード等） |
| ------------ | -------------- | ----------------------------------------- |
| <変更の要約> | <version>      | <検索キーワード>                          |

### 推移的依存の追加・削除・メジャーアップ

<!-- 該当がない場合: 「該当なし」と記載 -->

- <依存名> <旧> → <新>（<patch / minor / **major**>）— <備考>

### 差分取得不能の場合

<!-- 該当する場合のみ記載。それ以外はこのセクションごと省略する -->

- **理由**: <ソースリポジトリが GitHub 以外 / タグ命名が特定できない / CHANGELOG なし など>
- **試行内容**:
  - <試行 1>
  - <試行 2>
- **`/review` 側への申し送り**: コードベースで `<package>` の利用箇所を grep して影響範囲を広めに確認してください。
```

### ステップ 10: 完了報告

呼び出し元のスキル（`/review-pr` や `/fix-dependabot-alert`）に以下を報告する:

```
## 差分サマリー追記完了

**対象**: <package>@<old_version> → <new_version> (<ecosystem>)
**追記先**: <output_path>
**取得結果**: <CHANGELOG / GitHub Releases / commit log / 差分取得不能>
**抽出件数**: CVE <N>件 / Breaking change <N>件 / 推移的依存 <N>件
```

## 補足

### 大きすぎる CHANGELOG への対処

`gh api compare` のレスポンスは GitHub の制限で `.files[].patch` が省略されることがある（>3000 行など）。その場合は次の代替手段を試す:

1. `gh api repos/<owner>/<repo>/contents/CHANGELOG.md?ref=<new_tag>` で `new_tag` の CHANGELOG を取得し、`old_version` のセクションヘッダ（例: `## 6.0.1`）までを切り出す
2. それでもうまくいかない場合は GitHub Releases にフォールバック

### モノレポでのタグ命名

Rails のように `v8.0.4.1` という単一タグで複数 component（actionview, activerecord 等）の CHANGELOG を持つケースがある。この場合は `.files[]` で抽出された複数の CHANGELOG を component ごとに整理し、「出典バージョン」列に component 名を含めて区別する（例: `actionview@8.0.4.1`）。

### 文字列ベースのバージョン比較

`gh api releases --jq 'select(.tag_name >= "<old>")'` のような jq 比較は文字列比較なので、`v1.10` と `v1.9` の順序が逆転することがある。バージョン数が多くて並びが不安な場合は、`compare` API の `.commits[].commit.message` で間に含まれる全コミットを取得してから tag に紐付ける方が確実。簡易ケース（数バージョン以内）では文字列比較で十分。
