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

// generatorStatementは、生成器の代わりに置いたスタブが実行する文。スタブが
// 自身の文を書くのは、生成がトランザクションの内側で、クリーンアップの後、コミットの
// 前に実行されることをモックに固定させるため。トランザクションの外へ出された生成や、
// 書き込んだものを空にするクリーンアップより前に実行される生成は、黙って通過するのでは
// なく想定外の呼び出しとしてここに現れる。
const generatorStatement = `SELECT 'generated'`

// stubbedAccountsは、スタブした生成が作成したと報告するアカウント。実行が
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

// stubGenerateDataは生成器の代わりに立つ。
func stubGenerateData(ctx context.Context, tx *sql.Tx, _ *userRoster) ([]seedAccount, error) {
	if _, err := tx.ExecContext(ctx, generatorStatement); err != nil {
		return nil, err
	}

	return stubbedAccounts, nil
}

// TestRunner_Runは、破壊的なTRUNCATEを実データベースへ実行せずに、
// データベース操作の順序とトランザクション境界を検証する。
func TestRunner_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// environmentは実行が見るAPP_ENV。空はdevを意味し、ガード自身の
		// ケース以外はすべてその下で実行される。
		environment string
		// generateは生成器の代わりに立つもの。空は、generatorStatementを
		// 実行しstubbedAccountsを報告するスタブを意味する。
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
			// 失敗した生成は、クリーンアップをコミットしないまま残している。
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
			// ガードはTRUNCATEと、実行が辿り着くはずのなかったデータベースと
			// の間に立つものであるため、接続へ何かを尋ねるより前に効いている必要が
			// ある。ここで期待値を1つも登録しないのは意図的である。モックは期待する
			// よう告げられていない呼び出しをすべて拒否するため、Runから取り除かれた
			// ガードや、守るべき処理より後ろへ移されたガードは、黙って通過するのでは
			// なく異なるエラーとして表面化する。
			name:        "dev以外ではデータベースへ触れずに拒否する",
			environment: "prod",
			expect:      func(sqlmock.Sqlmock) {},
			wantErr:     "APP_ENV=prodでは実行できません",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("SQLモックの作成に失敗: %v", err)
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
				t.Fatalf("Runner.Run()のエラー = %v、%qを含むことを期待", err, tt.wantErr)
			}

			hasAccounts := strings.Contains(out.String(), accountsHeading)
			if hasAccounts != tt.wantReport {
				t.Errorf(
					"Runner.Run()のアカウント一覧の出力有無 = %t、期待値 = %t。出力 = %q",
					hasAccounts,
					tt.wantReport,
					out.String(),
				)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("満たされていないSQLの期待値がある: %v", err)
			}
		})
	}
}
