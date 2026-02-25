# Go版テスト実行速度の改善 設計書

<!--
このテンプレートの使い方:
1. このファイルを `docs/designs/2_todo/` ディレクトリにコピー
   例: cp docs/designs/template.md docs/designs/2_todo/new-feature.md
2. [機能名] などのプレースホルダーを実際の内容に置き換え
3. 各セクションのガイドラインに従って記述
4. コメント（ `\<!-- ... --\>` ）はガイドラインとして残しておくことを推奨

**公開時の注意事項**:
- 開発用ドメイン名を記載する場合は `example.dev` を使用してください（実際のドメイン名は記載しない）
- 環境変数の値はサンプル値のみ記載し、実際の値は含めないでください
-->

## 実装ガイドラインの参照

<!--
**重要**: 設計書を作成する前に、対象プラットフォームのガイドラインを必ず確認してください。
特に以下の点に注意してください：
- ディレクトリ構造・ファイル名の命名規則
- コーディング規約
- アーキテクチャパターン

ガイドラインに沿わない設計は、実装時にそのまま実装されてしまうため、
設計書作成の段階でガイドラインに準拠していることを確認してください。
-->

### Go版の実装の場合

以下のガイドラインに従って設計・実装を行ってください：

- [@go/CLAUDE.md](/workspace/go/CLAUDE.md) - 全体的なコーディング規約
- [@go/docs/architecture-guide.md](/workspace/go/docs/architecture-guide.md) - アーキテクチャガイド

## 概要

<!--
ガイドライン:
- この機能が「何を」実現するのかを簡潔に説明
- ユーザーにとっての価値や背景を記述
- 2-3段落程度で簡潔に
-->

Go版のテスト実行速度を改善するために、テスト基盤を最適化します。対象は **Wikino** (`/workspace/go/`)、**Annict** (`/annict/go/`)、**Mewst** (`/mewst/go/`) の3プロジェクトです。現在、各テストケースごとにDB接続の作成・テストデータのINSERT・ロールバックを行っていますが、この繰り返しがテスト数の増加に伴いボトルネックになりつつあります。

CIでの `-short` フラグによる短時間テスト限定実行の廃止、DB接続のプール化、テスト用bcryptコストの低減、共有fixturesの導入を段階的に実施し、テスト実行速度を大幅に改善します。

**目的**:

- テストのフィードバックループを短縮し、開発効率を向上させる
- テスト数が増加しても実行時間が線形に増加しない基盤を構築する

**背景**:

- 3プロジェクトすべてでCIの `-short` フラグと `testing.Short()` の使い方に課題がある
- パスワード関連テストでbcrypt DefaultCost (10) を使用しており、ハッシュ1回あたり約100msのコスト
- テストケースごとにビルダーでINSERTするため、同じようなデータを何度も作成している

**プロジェクト別の現状**:

| 項目                   | Wikino                                   | Annict                          | Mewst                                    |
| ---------------------- | ---------------------------------------- | ------------------------------- | ---------------------------------------- |
| テスト数               | 596                                      | -                               | -                                        |
| DB接続プール化         | ❌ 未実装                                | ✅ `sync.Once` で実装済み       | ❌ 未実装                                |
| `testing.Short()` 使用 | ❌ 未使用（`-short` が無意味）           | seedテスト2ファイルのみ         | `SetupTestDB` 内（全DBテストをスキップ） |
| CIでのDBテスト         | ✅ 実行される（`-short` が無意味のため） | ✅ 実行される（seedテスト以外） | ❌ **全DBテストがスキップされている**    |
| bcryptコスト           | DefaultCost (10)                         | DefaultCost (10)                | DefaultCost (10)                         |
| テストごとのDB接続     | `sql.Open` + `Ping` + `Close`            | `Begin` + `Rollback` のみ       | `sql.Open` + `Ping` + `Close`            |

## 要件

<!--
ガイドライン:
- 機能要件: 「何ができるべきか」を記述
- 非機能要件: 「どのように動くべきか」を必要に応じて記述
-->

### 機能要件

- テスト基盤を改善し、テスト実行速度を短縮する
- 既存テストの動作を壊さない（後方互換性を維持）
- テストの独立性と並列実行の安全性を維持する
- 既存のビルダーパターンと併用できる設計にする

### 非機能要件

- **パフォーマンス**: 全テストの実行時間を現状から50%以上短縮することを目標とする
- **保守性**: fixturesの追加・変更が容易であること。スキーマ変更時の影響が最小限であること
- **可読性**: テストコードから「どのデータを前提にしているか」が明確に読み取れること

## 設計

### 現状の分析（Wikino）

#### テスト実行時間の内訳（キャッシュなし）

| パッケージ               | テスト数  | 実行時間 | 推定ボトルネック                       |
| ------------------------ | --------- | -------- | -------------------------------------- |
| `usecase`                | 26        | 5.85s    | bcrypt + DB接続 + トランザクション管理 |
| `handler/sign_in`        | 28        | 4.86s    | bcrypt + DB接続                        |
| `handler/password_reset` | 23        | 4.31s    | bcrypt + DB接続                        |
| `auth`                   | 17        | 3.95s    | bcrypt（DB不使用）                     |
| `handler/account`        | 38        | 3.21s    | bcrypt + DB接続                        |
| `turnstile`              | 9         | 3.01s    | HTTPモック                             |
| その他                   | 各1〜1.6s | -        | コンパイル + 基本オーバーヘッド        |

#### 現状の `SetupTestDB` の動作

```go
func SetupTestDB(t *testing.T) (*sql.DB, *sql.Tx) {
    db, err := sql.Open("postgres", dbURL)  // 毎回接続作成
    db.Ping()                                // 毎回接続確認
    tx, err := db.Begin()                    // トランザクション開始
    t.Cleanup(func() {
        tx.Rollback()
        db.Close()                           // 毎回接続クローズ
    })
    return db, tx
}
```

### 現状の分析（Annict）

- **DB接続プール化**: `sync.Once` パターンで既に実装済み。テストごとの `sql.Open` は不要
- **`testing.Short()`**: seedテスト2ファイルでのみ使用（`create_user_test.go`, `create_work_test.go`）
- **bcrypt**: `auth/password.go` と `auth/verification_code.go` で `bcrypt.DefaultCost` を使用
- **Makefile**: Wikinoと同じCI分岐パターン。CIでは `-short` フラグ付きで実行

### 現状の分析（Mewst）

- **DB接続プール化**: 未実装。テストごとに `sql.Open` + `Ping` + `db.Close()` を実行
- **`testing.Short()`**: `SetupTestDB` 内で使用。CIの `-short` フラグにより**全DBテストがスキップされている**（重大な問題）
- **bcrypt**: `auth/password.go` で `bcrypt.DefaultCost` を使用
- **Makefile**: Wikinoと同じCI分岐パターン

### 改善方針

Wikinoでは4つの施策を段階的に実施します。Annict・Mewstはプロジェクトの現状に応じた施策を実施します。

#### 施策0: CIでの `-short` フラグ廃止とMakefile簡素化

現在、Makefileの `test` ターゲットでは `CI` 環境変数によってテスト実行コマンドを分岐しています：

```makefile
# 現状（Before）
test: db-setup-test
	@if [ "$$CI" = "true" ]; then \
		echo "CI環境: 短時間テストのみ実行 (-short)"; \
		bash -c 'set -o pipefail && go test -json -race -short ./... | tparse'; \
	else \
		echo "ローカル環境: 全テストを実行"; \
		bash -c 'set -o pipefail && APP_ENV=test op run --env-file=".env" -- go test -json -race ./... | tparse'; \
	fi
```

**問題点**:

- `testing.Short()` を使用しているテストが1つも存在しないため、`-short` フラグは実質的に何もスキップしていない
- テスト速度が改善されればCIでも全件テストを実行すべき
- CI/ローカルで異なるテストが実行されるのは信頼性の観点で望ましくない

**改善後（After）**:

```makefile
# 改善後: CI分岐を削除し、-shortフラグを廃止
test: db-setup-test
	@which tparse > /dev/null || (echo "Installing tparse from go.mod..." && go install github.com/mfridman/tparse)
	@bash -c 'set -o pipefail && go test -json -race ./... | tparse'
```

**変更点**:

- `CI` 環境変数による分岐を削除
- `-short` フラグを削除
- CI・ローカルともに同じコマンドで全テストを実行
- `op run --env-file=".env"` はCIでは不要（GitHub Actionsで環境変数を直接設定するため）、ローカルでは `make test` 実行前に環境変数が設定済みである前提とする

**`db-setup-test` ターゲットの変更**:

`db-setup-test` でもCI分岐（PostgreSQLホスト名の切り替え）がありますが、これはそのまま維持します。CIとローカルではPostgreSQLのホスト名が異なるためです。

#### 施策1: DB接続プールの共有化（TestMain パターン）

各テストパッケージで `TestMain` を使用し、DB接続を1回だけ確立してパッケージ内の全テストで共有します。

**Before（現状）**:

```
テスト1: sql.Open → Ping → Begin → テスト実行 → Rollback → Close
テスト2: sql.Open → Ping → Begin → テスト実行 → Rollback → Close
テスト3: sql.Open → Ping → Begin → テスト実行 → Rollback → Close
```

**After（改善後）**:

```
TestMain: sql.Open → Ping
テスト1: Begin → テスト実行 → Rollback
テスト2: Begin → テスト実行 → Rollback
テスト3: Begin → テスト実行 → Rollback
TestMain: Close
```

**実装**:

```go
// internal/testutil/db.go に追加

// testDB はパッケージレベルで共有するDB接続プール
var testDB *sql.DB

// SetupTestMain はTestMain内で呼び出し、パッケージ共有のDB接続を初期化する
func SetupTestMain(m *testing.M) int {
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        dbURL = "postgres://postgres:postgres@postgresql:5432/wikino_test?sslmode=disable"
    }

    var err error
    testDB, err = sql.Open("postgres", dbURL)
    if err != nil {
        slog.Error("テスト用DB接続に失敗", "error", err)
        return 1
    }
    defer testDB.Close()

    if err := testDB.Ping(); err != nil {
        slog.Error("テスト用DBへのpingに失敗", "error", err)
        return 1
    }

    return m.Run()
}

// SetupTx はテスト用のトランザクションをセットアップする
// SetupTestMainで初期化されたDB接続プールを使用する
func SetupTx(t *testing.T) (*sql.DB, *sql.Tx) {
    t.Helper()

    if testDB == nil {
        t.Fatal("SetupTestMainが呼ばれていません。TestMain内でtestutil.SetupTestMain(m)を呼んでください")
    }

    tx, err := testDB.Begin()
    if err != nil {
        t.Fatalf("トランザクション開始に失敗: %v", err)
    }

    t.Cleanup(func() {
        _ = tx.Rollback()
    })

    return testDB, tx
}
```

**各テストパッケージでの使用**:

```go
// internal/handler/sign_in/create_test.go

func TestMain(m *testing.M) {
    os.Exit(testutil.SetupTestMain(m))
}

func TestCreate_Success(t *testing.T) {
    t.Parallel()

    // SetupTestDB の代わりに SetupTx を使用
    db, tx := testutil.SetupTx(t)
    // 以降は既存のテストコードと同じ
}
```

**後方互換性**: 既存の `SetupTestDB` はそのまま残し、新しい `SetupTx` を追加します。既存テストは段階的に移行できます。

#### 施策2: テスト用bcryptコストの低減

テスト環境では bcrypt コストを最小値 (4) に設定し、パスワードハッシュの計算時間を大幅に削減します。

**実装**:

```go
// internal/auth/password.go

// BcryptCost はbcryptのコスト値。テスト時はTestBcryptCostに変更される
var BcryptCost = bcrypt.DefaultCost // 10

// TestBcryptCost はテスト用の低コスト値
const TestBcryptCost = bcrypt.MinCost // 4

// HashPassword はパスワードをbcryptでハッシュ化します。
func HashPassword(plainPassword string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), BcryptCost)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}
```

```go
// internal/testutil/db.go に追加

func SetupTestMain(m *testing.M) int {
    // テスト用にbcryptコストを下げる
    auth.BcryptCost = auth.TestBcryptCost

    // ... DB接続の初期化
}
```

**効果**: bcrypt cost 10 → 4 で約64倍高速化（2^10 / 2^4 = 64）。パスワードハッシュ1回あたり約100ms → 約1.5msに短縮。

#### 施策3: 共有fixturesの導入（SAVEPOINT パターン）

頻繁に使用するテストデータ（ユーザー、パスワードなど）を `TestMain` で1回だけINSERTし、各テストはSAVEPOINTで分離することで、テストごとのデータ作成コストを削減します。

**仕組み**:

```
TestMain:
  1. DB接続確立
  2. 共有fixturesをINSERT（共通のユーザー、セッションなど）

テスト1:
  1. SAVEPOINT sp_test1
  2. テスト実行（共有fixturesを参照 + 追加データをINSERT）
  3. ROLLBACK TO SAVEPOINT sp_test1 → テスト1の変更のみ取り消し

テスト2:
  1. SAVEPOINT sp_test2
  2. テスト実行
  3. ROLLBACK TO SAVEPOINT sp_test2

TestMain:
  4. 共有fixturesをDELETE（またはトランザクション全体をROLLBACK）
```

**重要な制約**: `t.Parallel()` との互換性のため、共有fixturesは**トランザクション外**（直接DBに）INSERTし、各テストは独立したトランザクション内で実行します。テスト終了後にfixturesをDELETEします。

**実装**:

```go
// internal/testutil/fixtures.go

// Fixtures はテスト間で共有するテストデータを保持する
type Fixtures struct {
    // 共通のテストユーザー（パスワード: "testpassword123"）
    UserID            string
    UserEmail         string
    UserAtname        string
    UserPasswordDigest string

    // 認証済みセッション
    SessionToken string
}

// SetupFixtures は共有fixturesをDBにINSERTする
// TestMain内で1回だけ呼び出す
func SetupFixtures(db *sql.DB) (*Fixtures, func()) {
    ctx := context.Background()
    fixtures := &Fixtures{}

    // パスワードハッシュ（テスト用低コスト）
    passwordDigest, _ := auth.HashPassword("testpassword123")
    fixtures.UserPasswordDigest = passwordDigest
    fixtures.UserEmail = "fixture-user@example.com"
    fixtures.UserAtname = "fixture_user"

    // ユーザーをINSERT
    err := db.QueryRowContext(ctx,
        `INSERT INTO users (id, email, atname, created_at, updated_at)
         VALUES (generate_ulid(), $1, $2, NOW(), NOW())
         RETURNING id`,
        fixtures.UserEmail, fixtures.UserAtname,
    ).Scan(&fixtures.UserID)
    if err != nil {
        panic(fmt.Sprintf("fixtureユーザーの作成に失敗: %v", err))
    }

    // パスワードをINSERT
    _, err = db.ExecContext(ctx,
        `INSERT INTO user_passwords (id, user_id, password_digest, created_at, updated_at)
         VALUES (generate_ulid(), $1, $2, NOW(), NOW())`,
        fixtures.UserID, passwordDigest,
    )
    if err != nil {
        panic(fmt.Sprintf("fixtureパスワードの作成に失敗: %v", err))
    }

    // セッションをINSERT
    token := "fixture-session-token-for-testing"
    _, err = db.ExecContext(ctx,
        `INSERT INTO user_sessions (id, user_id, token, ip_address, user_agent, created_at, updated_at)
         VALUES (generate_ulid(), $1, $2, '127.0.0.1', 'TestAgent', NOW(), NOW())`,
        fixtures.UserID, token,
    )
    if err != nil {
        panic(fmt.Sprintf("fixtureセッションの作成に失敗: %v", err))
    }
    fixtures.SessionToken = token

    // クリーンアップ関数
    cleanup := func() {
        db.ExecContext(ctx, "DELETE FROM user_sessions WHERE user_id = $1", fixtures.UserID)
        db.ExecContext(ctx, "DELETE FROM user_passwords WHERE user_id = $1", fixtures.UserID)
        db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", fixtures.UserID)
    }

    return fixtures, cleanup
}
```

**テストでの使用例**:

```go
var fixtures *testutil.Fixtures

func TestMain(m *testing.M) {
    code := testutil.SetupTestMain(m)

    var cleanup func()
    fixtures, cleanup = testutil.SetupFixtures(testutil.GetTestDB())
    defer cleanup()

    os.Exit(code)
}

func TestCreate_Success(t *testing.T) {
    t.Parallel()

    db, tx := testutil.SetupTx(t)

    // 共有fixturesのユーザーを使用（INSERTなし）
    // fixtures.UserEmail, fixtures.UserPasswordDigest を参照

    // テスト固有のデータはビルダーで作成
    // testutil.NewPasswordResetTokenBuilder(t, tx).
    //     WithUserID(fixtures.UserID).
    //     Build()
}
```

**共有fixturesとビルダーの使い分け**:

| データ種別                           | 方法                       | 理由                                     |
| ------------------------------------ | -------------------------- | ---------------------------------------- |
| 認証済みユーザー（ログインテスト用） | 共有fixtures               | 多くのテストで同じデータが必要           |
| パスワードリセットトークン           | ビルダー                   | テストケースごとに状態が異なる           |
| 「ユーザーが存在しない」テスト       | ビルダー不使用             | fixturesは使わず、存在しないメールで検証 |
| 2FA設定済みユーザー                  | 共有fixtures（別ユーザー） | 2FA関連テストで頻繁に使用                |

### 並列テストとの整合性

**重要**: `t.Parallel()` を使用する場合、各テストは独立したトランザクションで実行されます。

- **共有fixtures**: トランザクション外（直接DB）にINSERTされるため、全テストから参照可能
- **テスト固有データ**: 各テストのトランザクション内でINSERTされるため、他テストには見えない
- **書き込みテスト**: 共有fixturesのデータを変更する場合は、トランザクション内で行うためロールバックで元に戻る

```
DB状態: [共有fixtures: ユーザーA, セッションA]

テスト1 (tx1): ユーザーAを参照 + ユーザーBをINSERT → Rollback（ユーザーBは消える、ユーザーAは残る）
テスト2 (tx2): ユーザーAを参照 + ユーザーAのパスワードを更新 → Rollback（更新は取り消される）
テスト3 (tx3): 存在しないメールでログイン → Rollback
```

## タスクリスト

<!--
ガイドライン:
- フェーズごとに段階的な実装計画を記述
- チェックボックスで進捗を管理
- **重要**: 1タスク = 1 Pull Request の粒度で作成してください
- **重要**: 各タスクには想定ファイル数と想定行数を明記してください（PRサイズの見積もりのため）
- 想定ファイル数は「実装」と「テスト」に分けて記載してください
- 想定行数も「実装」と「テスト」に分けて記載してください
- 依存関係を明確に
- Pull Requestのガイドラインは CLAUDE.md を参照（変更ファイル数20以下、変更行数300行以下）

タスク番号の付け方:
- 各タスクには階層的な番号を付与します（例: 1-1, 1-2, 2-1, 2-2）
- フォーマット: **フェーズ番号-タスク番号**: タスク名
- **フェーズ番号は半角英数字とハイフンのみで表記**してください（ブランチ名に使用するため）
  - 例: フェーズ 1, フェーズ 2, フェーズ 5a（フェーズ 5 と 6 の間に追加する場合）
  - NG: フェーズ 5.5（ドットは使用不可）
- タスクの前に別のタスクを追加する場合は、サブ番号を使用します
  - 例: タスク 2-1 の前にタスクを追加する場合 → 2-0
  - 例: タスク 2-0 の前にタスクを追加する場合 → 2-0-1
- この番号はブランチ名の一部として使用されます（例: feature-1-1, feature-2-0）

プラットフォームプレフィックス:
- Go版またはRails版の修正を行うタスクには、タスク名の先頭にプラットフォームを示すプレフィックスを付けてください
- フォーマット: **フェーズ番号-タスク番号**: [Go] タスク名 または **フェーズ番号-タスク番号**: [Rails] タスク名
- Go版とRails版の両方を修正する場合は、別々のタスクに分けてください
- 例:
  - `- [ ] **1-1**: [Go] マイグレーション作成`
  - `- [ ] **1-2**: [Rails] モデルへのコールバック追加`
-->

### フェーズ 0: CIテスト実行の統一

<!--
テスト速度改善の前提として、CIとローカルで同じテストが実行されるようにする。
-short フラグは testing.Short() を使用しているテストが存在しないため実質無意味。
-->

- [x] **0-1**: [Go] Makefileの `test` ターゲットからCI分岐と `-short` フラグを削除
  - `test` ターゲットの `if [ "$$CI" = "true" ]` 分岐を削除
  - `-short` フラグを削除し、CI・ローカルで統一されたテスト実行コマンドにする
  - `op run --env-file=".env"` のラッパーを削除（環境変数は実行前に設定済みの前提）
  - `db-setup-test` のCI分岐（PostgreSQLホスト名）はそのまま維持
  - CIが正常に動作することを確認
  - **想定ファイル数**: 約 1 ファイル（実装 1）
  - **想定行数**: 約 10 行（実装 10 行）

### フェーズ 1: DB接続プール化とbcryptコスト最適化

<!--
最も効果が大きく、リスクの低い施策を先に実施する。
既存テストの書き方を大きく変更せずに速度改善が得られる。
-->

- [x] **1-1**: [Go] テスト用DB接続プール化（TestMainパターン）の導入
  - `testutil.SetupTestMain(m *testing.M)` を追加
  - `testutil.SetupTx(t *testing.T)` を追加（プール化されたDBからトランザクションを取得）
  - `testutil.GetTestDB()` を追加（プール化されたDBへの参照を取得）
  - 既存の `SetupTestDB` はそのまま残す（後方互換性）
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 100 行（実装 60 行 + テスト 40 行）

- [x] **1-2**: [Go] テスト用bcryptコストの低減
  - `auth.BcryptCost` 変数を導入（デフォルト: `bcrypt.DefaultCost`）
  - `auth.TestBcryptCost` 定数を追加（値: `bcrypt.MinCost`）
  - `HashPassword` を `BcryptCost` 変数を使用するように修正
  - `testutil.SetupTestMain` 内でテスト用コストに設定
  - 既存テストが壊れないことを確認
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 50 行（実装 30 行 + テスト 20 行）

- [x] **1-3**: [Go] 既存テストをTestMainパターンに移行（handlerパッケージ）
  - `handler/sign_in`, `handler/password_reset`, `handler/account` に `TestMain` を追加
  - `SetupTestDB` → `SetupTx` に置き換え
  - テストが正常に動作することを確認
  - **想定ファイル数**: 約 6 ファイル（実装 6 + テスト 0）
  - **想定行数**: 約 60 行（実装 60 行 + テスト 0 行）

- [x] **1-4**: [Go] 残りのテストをTestMainパターンに移行
  - `handler/password`, `handler/email_confirmation`, `handler/sign_in_two_factor` 等に `TestMain` を追加
  - `usecase`, `repository` パッケージのテストも移行
  - `SetupTestDB` を非推奨にする（コメントで案内）
  - **想定ファイル数**: 約 10 ファイル（実装 10 + テスト 0）
  - **想定行数**: 約 100 行（実装 100 行 + テスト 0 行）

- [x] **1-5**: [Go] 非推奨テストヘルパー (`SetupTestDB`, `SetupTestDBWithoutTx`) の削除
  - タスク 1-4 で全テストが `SetupTx` / `GetTestDB` に移行済みのため、非推奨メソッドを削除可能
  - `internal/testutil/db.go` から `SetupTestDB` と `SetupTestDBWithoutTx` を削除
  - `internal/testutil/db_test.go` から関連テストを削除
  - `go/CLAUDE.md` のテストヘルパー「非推奨」セクションを削除
  - `go/docs/architecture-guide.md`, `go/docs/templ-guide.md`, `go/docs/validation-guide.md` のコード例を `SetupTx` パターンに更新
  - **想定ファイル数**: 約 6 ファイル（実装 2 + ドキュメント 4）
  - **想定行数**: 約 80 行（実装 20 行削除 + ドキュメント 60 行修正）

### フェーズ 2: 共有fixturesの導入

<!--
フェーズ1で十分な速度改善が得られるか計測した上で、
必要であればフェーズ2を実施する。
-->

- [ ] **2-1**: [Go] Fixtures基盤の実装
  - `testutil.Fixtures` 構造体の定義
  - `testutil.SetupFixtures(db)` 関数の実装（共通テストデータのINSERT）
  - クリーンアップ関数の実装
  - Fixturesの動作テスト
  - **想定ファイル数**: 約 2 ファイル（実装 1 + テスト 1）
  - **想定行数**: 約 150 行（実装 100 行 + テスト 50 行）

- [ ] **2-2**: [Go] sign_inテストをFixturesパターンに移行（パイロット）
  - `handler/sign_in` の `TestMain` にFixturesセットアップを追加
  - 認証成功テストをFixturesのユーザーを使用するように変更
  - ビルダーで作成していた共通ユーザーをFixturesに置き換え
  - テスト固有のデータは引き続きビルダーで作成
  - 移行前後のテスト実行時間を計測・比較
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 100 行（実装 80 行 + テスト 20 行）

- [ ] **2-3**: [Go] 他のhandlerテストをFixturesパターンに移行
  - `handler/password_reset`, `handler/account`, `handler/password` 等を移行
  - 共通パターンのドキュメントを `go/CLAUDE.md` のテスト戦略セクションに追加
  - **想定ファイル数**: 約 8 ファイル（実装 6 + テスト 0 + ドキュメント 2）
  - **想定行数**: 約 150 行（実装 120 行 + ドキュメント 30 行）

### フェーズ 3: ドキュメント整備と計測

- [x] **3-1**: [Go] テスト基盤のドキュメント更新
  - `go/CLAUDE.md` のテスト戦略セクションを更新
  - `SetupTestMain` / `SetupTx` / Fixtures の使い方を記載
  - 新規テスト作成時のガイドライン追加
  - 計測結果（Before/After）を記載
  - **想定ファイル数**: 約 1 ファイル（実装 0 + テスト 0 + ドキュメント 1）
  - **想定行数**: 約 80 行（ドキュメント 80 行）

### フェーズ 4: Annict テスト改善

<!--
Annict (`/annict/go/`) のテスト基盤を改善する。
現状の分析:
- DB接続プール化: sync.Once パターンで既に実装済み（SetupTestDB内）
- testing.Short(): seedテスト2ファイルでのみ使用（create_user_test.go, create_work_test.go）
- Makefile: CI分岐と -short フラグあり（Wikinoと同じパターン）
- bcrypt: DefaultCost (10) を使用（auth/password.go）
- verification_code.go でもbcryptを使用
-->

- [x] **4-1**: [Go/Annict] Makefileの `test` ターゲットからCI分岐と `-short` フラグを削除
  - `/annict/go/Makefile` の `test` ターゲットの `if [ "$$CI" = "true" ]` 分岐を削除
  - `-short` フラグを削除し、CI・ローカルで統一されたテスト実行コマンドにする
  - `op run --env-file=".env"` のラッパーを削除
  - `db-setup-test` のCI分岐（PostgreSQLホスト名）はそのまま維持
  - **想定ファイル数**: 約 1 ファイル（実装 1）
  - **想定行数**: 約 10 行（実装 10 行）

- [x] **4-2**: [Go/Annict] seedテストの `testing.Short()` を削除
  - `internal/usecase/seed/create_user_test.go` の `testing.Short()` 分岐を削除
  - `internal/usecase/seed/create_work_test.go` の `testing.Short()` 分岐を削除
  - `-short` フラグが廃止されるため、条件分岐が不要になる
  - **想定ファイル数**: 約 2 ファイル（実装 2）
  - **想定行数**: 約 10 行（実装 10 行）

- [x] **4-3**: [Go/Annict] テスト用bcryptコストの低減
  - `internal/auth/password.go` の `HashPassword` を変数化されたコストで呼ぶように修正
  - `BcryptCost` 変数と `TestBcryptCost` 定数を追加
  - `verification_code.go` のbcrypt使用箇所も同様に修正
  - `SetupTestDB` の `sync.Once` 内でテスト用コストに設定（既にプール化されているため `TestMain` パターンは不要）
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 50 行（実装 30 行 + テスト 20 行）

- [ ] **4-4**: [Go/Annict] 共有fixturesの導入（必要に応じて）
  - フェーズ4-3まで実施した後の計測結果を踏まえて判断
  - Annict は既にDB接続プール化済みのため、bcryptコスト低減だけで十分な改善が得られる可能性が高い
  - **想定ファイル数**: 未定
  - **想定行数**: 未定

### フェーズ 5: Mewst テスト改善

<!--
Mewst (`/mewst/go/`) のテスト基盤を改善する。
現状の分析:
- DB接続プール化: 未実装（テストごとに sql.Open + Ping + Close）
- testing.Short(): SetupTestDB 内で使用 → CIの -short フラグで全DBテストがスキップされている
- Makefile: CI分岐と -short フラグあり（Wikinoと同じパターン）
- bcrypt: DefaultCost (10) を使用（auth/password.go）
- 重大な問題: CIでは -short フラグにより全DBテストがスキップされているため、
  DBを使うテストがCIで一切実行されていない
-->

- [x] **5-1**: [Go/Mewst] `SetupTestDB` から `testing.Short()` を削除
  - `/mewst/go/internal/testutil/db.go` の `testing.Short()` によるスキップを削除
  - Makefileの `test` ターゲットのCI分岐・`-short` フラグ・`op run` ラッパーは既に除去済みのため変更不要
  - `db-setup-test` のCI分岐（PostgreSQLホスト名）はそのまま維持
  - **想定ファイル数**: 約 1 ファイル（実装 1）
  - **想定行数**: 約 5 行（実装 5 行）

- [x] **5-2**: [Go/Mewst] DB接続プール化（sync.Onceパターン）の導入
  - `internal/testutil/db.go` の `SetupTestDB` を `sync.Once` パターンに変更（Annictの実装を参考）
  - テストごとの `sql.Open` + `Ping` + `db.Close()` を廃止
  - 接続プールの設定を追加（`SetMaxOpenConns`, `SetMaxIdleConns`）
  - トランザクションのロールバックのみ各テストで実行
  - **想定ファイル数**: 約 1 ファイル（実装 1）
  - **想定行数**: 約 40 行（実装 40 行）

- [x] **5-3**: [Go/Mewst] テスト用bcryptコストの低減
  - `internal/auth/password.go` の `HashPassword` を変数化されたコストで呼ぶように修正
  - `BcryptCost` 変数と `TestBcryptCost` 定数を追加
  - `SetupTestDB` の `sync.Once` 内でテスト用コストに設定
  - **想定ファイル数**: 約 3 ファイル（実装 2 + テスト 1）
  - **想定行数**: 約 50 行（実装 30 行 + テスト 20 行）

- [ ] **5-4**: [Go/Mewst] 共有fixturesの導入（必要に応じて）
  - フェーズ5-3まで実施した後の計測結果を踏まえて判断
  - **想定ファイル数**: 未定
  - **想定行数**: 未定

### 実装しない機能（スコープ外）

<!--
今回は実装しないが、将来的に検討する機能を明記
-->

以下の機能は今回の実装では**実装しません**：

- **テストのモック化**: 実DBを使ったテストの方針は変更しない。モックを導入すると統合テストの価値が下がるため
- **テスト用の専用DBスキーマ**: テスト用DBは引き続き `db/schema.sql` から作成する。テスト専用のスキーマ変更は行わない
- **テストデータのYAML/JSONファイル管理**: fixturesはGoコードで定義する。外部ファイルの読み込みは複雑性が増すため
- **`SetupTestDB` の即時削除**: ~~後方互換性のため非推奨コメントのみ追加し、既存コードの動作を維持する~~ → タスク 1-5 で削除予定（全テストの移行が完了したため）

## 参考資料

<!--
参考にしたドキュメント、記事、OSSプロジェクトなど
-->

- [Go testing パッケージ - TestMain](https://pkg.go.dev/testing#hdr-Main)
- [PostgreSQL SAVEPOINT](https://www.postgresql.org/docs/current/sql-savepoint.html)
- [bcrypt コスト値とパフォーマンス](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
