package seed

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// generatorStatement is what the stubbed generation executes in place of the
// generators. The stub writes a statement of its own so that the mock holds
// the generation to running inside the transaction, after the cleanup and
// before the commit: a generation moved outside the transaction, or run before
// the cleanup that would empty what it had written, surfaces here as an
// unexpected call rather than passing quietly.
//
// [Ja] generatorStatement は、生成器の代わりに置いたスタブが実行する文。スタブが
// 自身の文を書くのは、生成がトランザクションの内側で、クリーンアップの後、コミットの
// 前に実行されることをモックに固定させるため。トランザクションの外へ出された生成や、
// 書き込んだものを空にするクリーンアップより前に実行される生成は、黙って通過するのでは
// なく想定外の呼び出しとしてここに現れる。
const generatorStatement = `SELECT 'generated'`

// stubbedAccounts is what the stubbed generation reports having created. A run
// reports the accounts it created rather than the ones the roster listed, so
// the report has to be fed from here.
//
// [Ja] stubbedAccounts は、スタブした生成が作成したと報告するアカウント。実行が
// 報告するのは、名簿に挙がっていたアカウントではなく作成したアカウントであるため、
// 報告の元はここから与える。
var stubbedAccounts = []seedAccount{
	{
		roster: rosterUser{
			role:   roleMain,
			atname: "seeduser1",
			email:  "seeduser1@example.com",
			note:   "主な確認対象",
		},
	},
}

// stubGenerateData stands in for the generators.
//
// [Ja] stubGenerateData は生成器の代わりに立つ。
func stubGenerateData(ctx context.Context, tx *sql.Tx, _ *userRoster) ([]seedAccount, error) {
	if _, err := tx.ExecContext(ctx, generatorStatement); err != nil {
		return nil, err
	}

	return stubbedAccounts, nil
}

// TestRunner_Run verifies the database operation order and transaction
// boundary without executing the destructive TRUNCATE against a real database.
//
// [Ja] TestRunner_Run は、破壊的な TRUNCATE を実データベースへ実行せずに、
// データベース操作の順序とトランザクション境界を検証する。
func TestRunner_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// environment is the APP_ENV the run sees. Empty means dev, which is
		// what every case but the guard's own is run under.
		//
		// [Ja] environment は実行が見る APP_ENV。空は dev を意味し、ガード自身の
		// ケース以外はすべてその下で実行される。
		environment string
		// generate stands in for the generators. Empty means the stub that
		// executes generatorStatement and reports stubbedAccounts.
		//
		// [Ja] generate は生成器の代わりに立つもの。空は、generatorStatement を
		// 実行し stubbedAccounts を報告するスタブを意味する。
		generate   func(ctx context.Context, tx *sql.Tx, roster *userRoster) ([]seedAccount, error)
		expect     func(sqlmock.Sqlmock)
		wantErr    string
		wantReport bool
	}{
		{
			name: "正常系では接続先を確認してクリーンアップと生成をコミットする",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).
					WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("mewst_dev"))
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(cleanupStatement())).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta(generatorStatement)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantReport: true,
		},
		{
			// A generation that failed has left the cleanup uncommitted, and
			// the rollback is what keeps a developer from being handed a
			// database that was emptied and never filled back in.
			//
			// [Ja] 失敗した生成は、クリーンアップをコミットしないまま残している。
			// ロールバックは、空にされたきり埋め直されていないデータベースを開発者へ
			// 渡さないためのもの。
			name: "生成失敗時はクリーンアップごとロールバックする",
			generate: func(context.Context, *sql.Tx, *userRoster) ([]seedAccount, error) {
				return nil, errors.New("アカウントの作成に失敗")
			},
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).
					WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("mewst_dev"))
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(cleanupStatement())).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectRollback()
			},
			wantErr: "アカウントの作成に失敗",
		},
		{
			name: "クリーンアップ失敗時はロールバックする",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).
					WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("mewst_dev"))
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(cleanupStatement())).WillReturnError(errors.New("cleanup failed"))
				mock.ExpectRollback()
			},
			wantErr: "既存データの削除に失敗",
		},
		{
			name: "コミット失敗時は成功報告を出さない",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).
					WillReturnRows(sqlmock.NewRows([]string{"current_database"}).AddRow("mewst_dev"))
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta(cleanupStatement())).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(regexp.QuoteMeta(generatorStatement)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			wantErr: "トランザクションのコミットに失敗",
		},
		{
			name: "接続先取得失敗時はトランザクションを開始しない",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT current_database()`)).WillReturnError(errors.New("query failed"))
			},
			wantErr: "接続先データベース名の取得に失敗",
		},
		{
			// The guard is what stands between the TRUNCATE and a database
			// the run was never meant to reach, so it has to hold before
			// anything at all is asked of the connection. No expectation is
			// registered here on purpose: the mock refuses every call it was
			// not told to expect, so a guard removed from Run, or moved below
			// the work it protects, surfaces as the wrong error rather than
			// passing quietly.
			//
			// [Ja] ガードは TRUNCATE と、実行が辿り着くはずのなかったデータベースと
			// の間に立つものであるため、接続へ何かを尋ねるより前に効いている必要が
			// ある。ここで期待値を 1 つも登録しないのは意図的である。モックは期待する
			// よう告げられていない呼び出しをすべて拒否するため、Run から取り除かれた
			// ガードや、守るべき処理より後ろへ移されたガードは、黙って通過するのでは
			// なく異なるエラーとして表面化する。
			name:        "dev 以外ではデータベースへ触れずに拒否する",
			environment: "prod",
			expect:      func(sqlmock.Sqlmock) {},
			wantErr:     "APP_ENV=prod では実行できません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("SQL モックの作成に失敗: %v", err)
			}
			t.Cleanup(func() { _ = db.Close() })
			tt.expect(mock)

			environment := tt.environment
			if environment == "" {
				environment = devEnvironment
			}

			var out bytes.Buffer
			runner := NewRunner(db, &out)
			runner.environment = func() string { return environment }
			runner.rosterPath = "../../" + rosterExamplePath
			runner.generateData = stubGenerateData
			if tt.generate != nil {
				runner.generateData = tt.generate
			}

			err = runner.Run(context.Background())

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Runner.Run() がエラーを返した: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Runner.Run() error = %v, want it to contain %q", err, tt.wantErr)
			}

			hasAccounts := strings.Contains(out.String(), accountsHeading)
			if hasAccounts != tt.wantReport {
				t.Errorf(
					"Runner.Run() account report = %t, want %t; output = %q",
					hasAccounts,
					tt.wantReport,
					out.String(),
				)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("満たされていない SQL の期待値がある: %v", err)
			}
		})
	}
}
