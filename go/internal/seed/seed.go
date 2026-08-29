// Package seed populates a development database with the data the screens
// cannot be looked at without: accounts to sign in as, posts to read, and the
// follows, reactions and exports that hang off them.
//
// The accounts themselves are configuration rather than code. Who exists is
// written in a roster file, and the code reaches for those accounts by role.
//
// A run empties the tables it manages before it generates, so that what a
// screen shows is always what the current code produces rather than what an
// earlier run left behind.
//
// [Ja] seed パッケージは、それが無いと画面を見られないデータ (サインインする
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

// appEnvVar is the environment variable that says which environment a process
// is running in.
//
// [Ja] appEnvVar は、プロセスがどの環境で動いているのかを示す環境変数。
const appEnvVar = "APP_ENV"

// devEnvironment is the only value of APP_ENV a run is allowed under.
//
// [Ja] devEnvironment は、実行を許可する唯一の APP_ENV の値。
const devEnvironment = "dev"

// rosterPath is the roster a run reads. Like the example beside it, it is
// relative to the directory a run is started in, which is the Go module root.
//
// [Ja] rosterPath は実行が読む名簿。隣に置いた見本と同じく、実行を開始した
// ディレクトリからの相対パスであり、それは Go モジュールのルートになる。
const rosterPath = "seed-users.toml"

// Runner rebuilds a development database from the roster and the generators.
//
// It takes the database handle and the stream to report itself on as fields
// rather than reaching for the process's own, so that what it writes to and
// what it writes into are both visible at the one place it is constructed.
//
// [Ja] Runner は、名簿と生成器から開発用データベースを作り直す。
//
// データベースハンドルと自身の報告先のストリームを、プロセス自身のものを取りに
// 行くのではなくフィールドで受け取る。何に書き込み、どこへ書き出すのかの双方を、
// 構築する 1 箇所で見えるようにするため。
type Runner struct {
	db          *sql.DB
	out         io.Writer
	environment func() string
	rosterPath  string

	// generateData writes the seed data into the transaction the cleanup has
	// already emptied. It is held as a field so that the transaction boundary
	// around it can be verified against a mock: the cleanup it runs after
	// empties every managed table, which is not a statement to send to the
	// database the rest of the test suite is working in.
	//
	// [Ja] generateData は、クリーンアップが空にし終えたトランザクションへシード
	// データを書き込む。フィールドで持つのは、その前後のトランザクション境界をモックに
	// 対して検証できるようにするため。直前に実行されるクリーンアップは管理対象の
	// テーブルをすべて空にするものであり、テストスイートの他の部分が作業中のデータ
	// ベースへ送ってよい文ではない。
	generateData func(ctx context.Context, tx *sql.Tx, roster *userRoster) ([]seedAccount, error)
}

// NewRunner returns a Runner that writes to db and reports itself on out.
//
// [Ja] NewRunner は、db へ書き込み、out へ自身を報告する Runner を返す。
func NewRunner(db *sql.DB, out io.Writer) *Runner {
	return &Runner{
		db:           db,
		out:          out,
		environment:  func() string { return os.Getenv(appEnvVar) },
		rosterPath:   rosterPath,
		generateData: generateSeedData,
	}
}

// Run empties the managed tables and generates the seed data in their place.
//
// The environment is checked first, before the roster is read and before
// anything is written, because everything that follows is destructive. Placing
// the check here rather than at the call site means it holds for every way the
// seed is reached, not only for the one subcommand that reaches it today.
//
// [Ja] Run は管理対象のテーブルを空にし、その場所へシードデータを生成する。
//
// 環境の検査を、名簿を読むより前、何かを書き込むより前の最初に行うのは、以降の
// すべてが破壊的であるため。この検査を呼び出し側ではなくここに置くことで、今日
// シードへ辿り着く唯一のサブコマンドだけでなく、シードへ辿り着くすべての経路に
// 対して検査が効く。
func (r *Runner) Run(ctx context.Context) error {
	if err := requireDevEnvironment(r.environment()); err != nil {
		return err
	}

	roster, err := loadUserRoster(r.rosterPath)
	if err != nil {
		return err
	}

	// The database is asked for its own name rather than read off the
	// connection string, which carries the password to it.
	//
	// [Ja] データベース名は接続文字列から読み取るのではなくデータベース自身に
	// 尋ねる。接続文字列はそこへのパスワードを持っているため。
	database, err := currentDatabase(ctx, r.db)
	if err != nil {
		return err
	}

	// What a run is pointed at is reported before it empties anything rather
	// than after. Every managed row is about to be discarded, and a report
	// that arrives afterwards names the database that was emptied to a reader
	// who can no longer choose otherwise.
	//
	// [Ja] 実行が何を向いているのかは、何かを空にした後ではなく前に報告する。
	// 管理対象の行はこれからすべて破棄されるのであり、後から届く報告は、もはや
	// 別の選択ができない読み手に対して、空にしてしまったデータベースの名前を
	// 告げることになる。
	progress := newProgress(r.out)
	progress.line("データベース %s を空にして、名簿 %s のアカウントから作り直します", database, roster.path)

	accounts, err := r.generate(ctx, roster)
	if err != nil {
		return err
	}

	progress.accounts(accounts)

	return nil
}

// generate empties the managed tables and runs the generators, in one
// transaction.
//
// The cleanup and the generation share a transaction so that a run that fails
// partway leaves the database as it was. A failure that had already committed
// the cleanup would leave a developer with a database that was emptied and
// never filled back in.
//
// [Ja] generate は、管理対象のテーブルを空にして生成器を実行する処理を、1 つの
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

// generateSeedData writes the seed data into tx, in the order the rows depend
// on each other: the application every post is attributed to, and then the
// accounts everything else hangs off.
//
// [Ja] generateSeedData は、行が互いに依存する順序でシードデータを tx へ書き込む。
// すべてのポストの帰属先となるアプリケーションを作り、次に、それ以外のすべてが
// ぶら下がるアカウントを作る。
func generateSeedData(ctx context.Context, tx *sql.Tx, roster *userRoster) ([]seedAccount, error) {
	if err := createOauthApplication(ctx, tx); err != nil {
		return nil, err
	}

	// One instant stands for the whole run. The accounts are placed relative
	// to it, and a run that read the clock once per account would place two of
	// them on either side of a month boundary depending on how long it took to
	// get from the one to the other.
	//
	// [Ja] 実行全体を 1 つの時点で代表させる。アカウントはその時点を基準に配置される。
	// アカウントごとに時計を読む実行は、一方から他方へ到達するまでにかかった時間に
	// よって、2 つのアカウントを月境界の両側へ置くことになる。
	return createAccounts(ctx, tx, roster, time.Now())
}

// requireDevEnvironment refuses a run outside a development environment.
//
// It is given the value rather than reading it, so that what it decides on is
// visible to a caller and to a test.
//
// An unset APP_ENV is refused along with a wrong one. config.Load reads an
// unset APP_ENV as dev, which is the right default for a process that only
// serves what it is asked for, but it would let a run that never named an
// environment empty whichever database DATABASE_URL happens to point at.
//
// [Ja] requireDevEnvironment は、開発環境以外での実行を拒否する。
//
// 値を自分で読まずに受け取るのは、何を見て判断しているのかを呼び出し側とテスト
// から見えるようにするため。
//
// 未設定の APP_ENV も、誤った値と同じく拒否する。config.Load は未設定の APP_ENV を
// dev として読み、それは求められたものを提供するだけのプロセスにとって正しい既定
// だが、環境を一度も名指ししなかった実行に、DATABASE_URL がたまたま指している
// データベースを空にさせることになる。
func requireDevEnvironment(env string) error {
	if env == "" {
		return fmt.Errorf(
			"%s が設定されていません。管理対象のテーブルをすべて空にするため、開発環境でだけ実行できます。%s=%s を明示してください",
			appEnvVar, appEnvVar, devEnvironment,
		)
	}
	if env != devEnvironment {
		return fmt.Errorf(
			"%s=%s では実行できません。管理対象のテーブルをすべて空にするため、開発環境 (%s=%s) でだけ実行できます",
			appEnvVar, env, appEnvVar, devEnvironment,
		)
	}

	return nil
}

// currentDatabase asks the connection which database it reached.
//
// [Ja] currentDatabase は、その接続がどのデータベースへ繋がったのかを尋ねる。
func currentDatabase(ctx context.Context, db *sql.DB) (string, error) {
	var name string

	if err := db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&name); err != nil {
		return "", fmt.Errorf("接続先データベース名の取得に失敗: %w", err)
	}

	return name, nil
}
