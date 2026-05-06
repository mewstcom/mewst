---
name: update-guidelines
description: APM 経由で配信されている korylus/guidelines を指定バージョンへ更新し、追加・削除・変更されたガイドラインに対する追従作業計画書を作成する。
argument-hint: "<version> (例: v1.2.0)"
---

# ガイドライン追従コマンド

`apm.yml` に記載されている `github.com/korylus/guidelines` を `$ARGUMENTS` で指定したバージョンに更新し、追加・削除・変更されたガイドラインに対してプロジェクト実装が追従できているかを並列調査して、未追従箇所を作業計画書として出力するコマンドです。

**指定バージョン**: $ARGUMENTS

## 前提

- カレントディレクトリがプロジェクトルート (`apm.yml` がある場所) であること
- `apm.yml` の `dependencies.apm` に `github.com/korylus/guidelines#vX.Y.Z` の形で依存が記載されていること
- 作業計画書テンプレートが `docs/private/plans/template.md` に存在すること
- 作業計画書の出力先 `docs/private/plans/2_todo/` が存在すること

## 手順

### ステップ 1: 入力検証と前提チェック

1. `$ARGUMENTS` が空でないことを確認する。空であれば「`/update-guidelines <version>` の形式で実行してください」と案内して中断する
2. `apm.yml` を読み、`dependencies.apm` に `github.com/korylus/guidelines#<version>` 形式のエントリが 1 件あることを確認する。無ければ「`apm.yml` に `github.com/korylus/guidelines` の依存が見つかりません」と案内して中断する
3. 現在のバージョンと `$ARGUMENTS` を比較し、同一であれば「現在のバージョンと同じため更新は不要です」と案内して中断する
4. `git ls-files .claude/rules .claude/skills` を実行し、結果が空であれば「`.claude/rules` と `.claude/skills` が git に追跡されていません。`apm install` を実行してこれらのファイルをコミットしてから再実行してください」と案内して中断する
5. `git status --porcelain .claude/rules .claude/skills` を実行し、未コミットの差分が無いことを確認する。差分があれば「`.claude/rules` と `.claude/skills` に未コミットの差分があります。先にコミットまたは退避してから再実行してください」と案内して中断する

### ステップ 2: バージョン更新と再インストール

1. `apm.yml` の `github.com/korylus/guidelines#<旧バージョン>` を `github.com/korylus/guidelines#$ARGUMENTS` に書き換える (Edit ツールで行単位の置換)
2. `apm install --update` をプロジェクトルートで実行する。失敗した場合は出力をユーザーに見せて中断する (バージョン取得失敗、apm.yml 構文エラーなど)

### ステップ 3: 差分の検出

1. 以下のコマンドで `.claude/rules` と `.claude/skills` の差分を取得する。`apm install --update` 直後なので、ステージング前 (Working Tree) の状態が直近の変更を表す:
   - 追加ファイル: `git ls-files --others --exclude-standard .claude/rules .claude/skills`
   - 削除ファイル: `git ls-files --deleted .claude/rules .claude/skills`
   - 変更ファイル: `git ls-files --modified .claude/rules .claude/skills`
   - 変更内容の取得 (Agent への入力用): 変更ファイルそれぞれに `git diff -- <path>` を実行

2. 検出した差分を以下のカテゴリに振り分ける:
   - **追加された rules**: `.claude/rules/*.md` で新たに作成されたファイル
   - **削除された rules**: `.claude/rules/*.md` で削除されたファイル
   - **変更された rules**: `.claude/rules/*.md` で内容が変わったファイル
   - **追加された skills**: `.claude/skills/*/` で新たに作成されたディレクトリ
   - **削除された skills**: `.claude/skills/*/` で削除されたディレクトリ

3. すべてのカテゴリが空であれば、「指定バージョンへの更新で `.claude/rules` と `.claude/skills` に差分は発生しませんでした。プロジェクト追従の必要はありません」と報告して中断する (作業計画書は作成しない)

### ステップ 4: 並列調査

差分があったガイドライン 1 件に対して 1 つの Agent (subagent_type: `general-purpose`) を立ち上げ、**全件を 1 つのメッセージにまとめて並列で起動**する。

各 Agent への指示テンプレート:

```
あなたは APM ガイドライン {ファイル名} の {追加|削除|変更} に対して、現在のプロジェクト実装が追従できているかを調査するエージェントです。

## ガイドラインの内容
{追加 or 変更の場合: 新ガイドラインの全文}
{削除の場合: 旧ガイドラインの全文}

## 変更内容 (変更ガイドラインのみ)
{git diff の出力}

## 調査対象
- プロジェクトルートはあなた (Agent) のカレントディレクトリと一致する
- ガイドラインの applyTo フロントマターに従ってコードベースを Grep / Read で調査する
- ガイドライン内の「ファイル名」「クラス名」「ディレクトリ構造」「規約」がどの程度プロジェクトに反映されているかを確認する
- 違反 / 未追従の箇所を列挙する。**修正方針が明確なものは併記する**

## 出力フォーマット
1. 違反・未追従の箇所を箇条書きで列挙する (該当ファイル / 行 / 違反内容を明記)
2. 各項目に対して推奨される修正内容を 1〜2 文で記述する
3. 「違反なし」「対応不要」と判断した場合はその旨だけを返す
4. 文字数の制限はない (全件列挙すること)
```

**注意**: Agent への入力にガイドライン全文を含めるため、ガイドラインファイルの内容は事前に Read しておく。

### ステップ 5: 作業計画書の作成

1. `docs/private/plans/template.md` をベースに `docs/private/plans/2_todo/update-guidelines-$ARGUMENTS.md` を作成する
2. テンプレートのプレースホルダーを以下の方針で埋める:
   - **タイトル**: `# korylus/guidelines $ARGUMENTS への追従 作業計画書`
   - **仕様書**: 「該当なし」
   - **概要**: 旧バージョン → `$ARGUMENTS` への更新と、検出した差分件数 (追加 N, 削除 N, 変更 N) を記載
   - **要件**: 「追加・変更された全ガイドラインに対してプロジェクト実装が追従していること」「削除されたガイドラインに紐付くプロジェクト固有の実装・記述を必要に応じてクリーンアップすること」
   - **実装ガイドラインの参照**: 該当する Go 版・Rails 版のガイドラインへのリンクをそのまま残す
   - **設計**: 「ガイドライン更新内容のサマリー」として差分ファイル一覧と要点を記載
   - **採用しなかった方針**: 「なし」(必要に応じて Agent の調査結果から記載)
   - **タスクリスト**:
     - **変更ガイドラインファイル 1 つ = フェーズ 1 つ** とする
     - フェーズ名: `[追加|削除|変更] {ガイドラインファイル名}` (例: `フェーズ 1: [変更] go-handler.md への追従`)
     - 各フェーズの中に、Agent が報告した違反・未追従項目を **1 項目 = 1 タスク** として配置する
     - タスク名は `[Go|Rails] <違反箇所の要約>` の形式 (両方に該当する場合は別タスクに分ける)
     - 各タスクには Agent の推奨修正内容をサブタスクとして展開
     - 想定ファイル数・行数は Agent の調査結果から見積もる (見積もれない場合は「未確定」と記載)
     - Agent が「違反なし」と返したガイドラインのフェーズは、タスクを置かずに「対応不要 (Agent 調査結果)」と注釈を記載する
   - **最終フェーズ**: テンプレート通り「仕様書への反映」を末尾に置く
   - **実装しない機能 (スコープ外)**: 「なし」
   - **参考資料**: korylus/guidelines のリリースノート (該当バージョンの GitHub リンク) を記載

3. テンプレートの HTML コメント (`<!-- ガイドライン -->`) は削除し、利用者向けの注釈だけ残す

### ステップ 6: ユーザーへの報告

以下を報告して終了する:

1. 旧バージョン → 新バージョンの差分概要 (追加 N 件、削除 N 件、変更 N 件)
2. 各カテゴリのファイル一覧
3. 作成した作業計画書のパス (`docs/private/plans/2_todo/update-guidelines-$ARGUMENTS.md`)
4. 「未コミットの変更を確認の上、`/commit` で `apm.yml` ・ `apm.lock.yaml` ・ `.claude/rules` ・ `.claude/skills` の更新と作業計画書をコミットしてください」というメッセージ
5. 「作業計画書のタスクは `docs/private/plans/1_doing/` に移動してから着手してください」というメッセージ

## 制約

- ブランチの作成、コミット、push は **行わない**
- AskUserQuestion は **使わない** (実行中の対話なし)
- Agent が「対応すべきか不明」と返した違反も、判断ごと作業計画書のタスクに残す (人間が後で取捨選択する)
- `apm install --update` 後、`apm.yml` ・ `apm.lock.yaml` ・ `.claude/rules` ・ `.claude/skills` の差分は **そのまま残す** (人間がコミット時にレビューできるように)
