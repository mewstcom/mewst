// Package seedは、それが無いと画面を見られないデータ (サインインする
// アカウント、読むためのポスト、それらにぶら下がるフォロー・リアクション・
// エクスポート) を開発用のデータベースへ投入する。
//
// アカウント自体はコードではなく設定とする。誰がいるのかは名簿ファイルに書かれて
// おり、コードはそのアカウントを役割で引く。
//
// 実行は生成の前に管理対象のテーブルを空にする。画面に出るものが常に現在のコードの
// 生成結果であり、前回の実行が残したものではないようにするため。
package seed

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"time"
)

// appEnvVarは、プロセスがどの環境で動いているのかを示す環境変数。
const appEnvVar = "APP_ENV"

// devEnvironmentは、実行を許可する唯一のAPP_ENVの値。
const devEnvironment = "dev"

// rosterPathは実行が読む名簿。隣に置いた見本と同じく、実行を開始した
// ディレクトリからの相対パスであり、それはGoモジュールのルートになる。
const rosterPath = "seed-users.toml"

// Runnerは、名簿と生成器から開発用データベースを作り直す。
//
// データベースハンドルと自身の報告先のストリームを、プロセス自身のものを取りに
// 行くのではなくフィールドで受け取る。何に書き込み、どこへ書き出すのかの双方を、
// 構築する1箇所で見えるようにするため。
type Runner struct {
	db          *sql.DB
	out         io.Writer
	environment func() string
	rosterPath  string

	// generateDataは、クリーンアップが空にし終えたトランザクションへシード
	// データを書き込む。フィールドで持つのは、その前後のトランザクション境界をモックに
	// 対して検証できるようにするため。直前に実行されるクリーンアップは管理対象の
	// テーブルをすべて空にするものであり、テストスイートの他の部分が作業中のデータ
	// ベースへ送ってよい文ではない。
	generateData func(ctx context.Context, tx *sql.Tx, roster *userRoster) ([]seedAccount, error)
}

// NewRunnerは、dbへ書き込み、outへ自身を報告するRunnerを返す。
func NewRunner(db *sql.DB, out io.Writer) *Runner {
	return &Runner{
		db:           db,
		out:          out,
		environment:  func() string { return os.Getenv(appEnvVar) },
		rosterPath:   rosterPath,
		generateData: generateSeedData,
	}
}

// Runは管理対象のテーブルを空にし、その場所へシードデータを生成する。
//
// 環境の検査を、名簿を読むより前、何かを書き込むより前の最初に行うのは、以降の
// すべてが破壊的であるため。この検査を呼び出し側ではなくここに置くことで、今日
// シードへ辿り着く唯一のサブコマンドだけでなく、シードへ辿り着くすべての経路に
// 対して検査が効く。
func (r *Runner) Run(ctx context.Context) error {
	if err := requireDevEnvironment(r.environment(), truncatesEveryManagedTable); err != nil {
		return err
	}

	roster, err := loadUserRoster(r.rosterPath)
	if err != nil {
		return err
	}

	// データベース名は接続文字列から読み取るのではなくデータベース自身に
	// 尋ねる。接続文字列はそこへのパスワードを持っているため。
	database, err := currentDatabase(ctx, r.db)
	if err != nil {
		return err
	}

	// 実行が何を向いているのかは、何かを空にした後ではなく前に報告する。
	// 管理対象の行はこれからすべて破棄されるのであり、後から届く報告は、もはや
	// 別の選択ができない読み手に対して、空にしてしまったデータベースの名前を
	// 告げることになる。
	progress := newProgress(r.out)
	progress.line("データベース%sを空にして、名簿%sのアカウントから作り直します", database, roster.path)

	accounts, err := r.generate(ctx, roster)
	if err != nil {
		return err
	}

	progress.accounts(accounts)

	return nil
}

// generateは、管理対象のテーブルを空にして生成器を実行する処理を、1つの
// トランザクションで行う。
//
// クリーンアップと生成でトランザクションを共有するのは、途中で失敗した実行が
// データベースを元のまま残すようにするため。クリーンアップをすでにコミットして
// いた失敗は、空にされたきり埋め直されていないデータベースを開発者に残す。
func (r *Runner) generate(ctx context.Context, roster *userRoster) ([]seedAccount, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := cleanup(ctx, tx); err != nil {
		return nil, err
	}

	accounts, err := r.generateData(ctx, tx, roster)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return accounts, nil
}

// generateSeedDataは、行が互いに依存する順序でシードデータをtxへ書き込む。
// すべてのポストの帰属先となるアプリケーションを作り、次に、それ以外のすべてが
// ぶら下がるアカウントを作り、次に、そのアカウントが書いたポストを作り、次に、
// さらに何件かのポストが持つリンクカードを作り、次に、フォローと、それらのポストが
// 埋めるタイムラインを作り、次に、それらのポストが受け取ったスタンプと、それが
// 起こした通知を作り、最後に、その時点で存在するポストのアーカイブを表すエクスポートを
// 作る。
func generateSeedData(ctx context.Context, tx *sql.Tx, roster *userRoster) ([]seedAccount, error) {
	applicationID, err := createOauthApplication(ctx, tx)
	if err != nil {
		return nil, err
	}

	// 実行全体を1つの時点で代表させる。すべてはその時点を基準に配置される。
	// 途中でもう一度時計を読む実行は、一方から他方へ到達するまでにかかった時間に
	// よって、2つの行を月境界の両側へ置くことになる。
	now := time.Now()

	accounts, err := createAccounts(ctx, tx, roster, now)
	if err != nil {
		return nil, err
	}

	if err := createPosts(ctx, tx, defaultAmounts, applicationID, accounts, now); err != nil {
		return nil, err
	}

	if err := createLinks(ctx, tx, applicationID, accounts, now); err != nil {
		return nil, err
	}

	if err := createFollows(ctx, tx, accounts, now); err != nil {
		return nil, err
	}

	if err := createReactions(ctx, tx, defaultAmounts, accounts, now); err != nil {
		return nil, err
	}

	if err := createExports(ctx, tx, accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// truncatesEveryManagedTableは、シードの実行を開発環境に限っている理由。
// 拒否がこれを報告することで、メッセージは、期待されていた値が何かだけでなく、
// 何が懸かっているのかを述べることになる。
const truncatesEveryManagedTable = "管理対象のテーブルをすべて空にするため"

// requireDevEnvironmentは、開発環境以外での実行を拒否する。
//
// 値を自分で読まずに受け取るのは、何を見て判断しているのかを呼び出し側とテスト
// から見えるようにするため。
//
// 未設定のAPP_ENVも、誤った値と同じく拒否する。config.Loadは未設定のAPP_ENVを
// devとして読み、それは求められたものを提供するだけのプロセスにとって正しい既定
// だが、環境を一度も名指ししなかった実行に、DATABASE_URLがたまたま指している
// データベースを空にさせることになる。
func requireDevEnvironment(env, reason string) error {
	if env == "" {
		return fmt.Errorf(
			"%sが設定されていません。%s、開発環境でだけ実行できます。%s=%sを明示してください",
			appEnvVar, reason, appEnvVar, devEnvironment,
		)
	}
	if env != devEnvironment {
		return fmt.Errorf(
			"%s=%sでは実行できません。%s、開発環境 (%s=%s) でだけ実行できます",
			appEnvVar, env, reason, appEnvVar, devEnvironment,
		)
	}

	return nil
}

// currentDatabaseは、その接続がどのデータベースへ繋がったのかを尋ねる。
func currentDatabase(ctx context.Context, db *sql.DB) (string, error) {
	var name string

	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return "", fmt.Errorf("接続先データベース名の取得に失敗: %w", err)
	}

	return name, nil
}
