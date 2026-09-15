// Package testutilはテスト用のヘルパー関数とビルダーを提供する
package testutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/mewstcom/mewst/go/internal/auth"
	"github.com/mewstcom/mewst/go/internal/query"
)

// mewstWebLockKeyはmewst-webのoauth_applications行を保護する
// advisory lockのキー (このアプリ内で一意な任意の固定値)。
const mewstWebLockKey int64 = 727577_001

var (
	testDB     *sql.DB
	testDBOnce sync.Once
)

// SetupTestMainはテストパッケージごとのTestMainで呼び出すヘルパー関数。
// bcryptコストの低減とDB接続プールの初期化をパッケージ内で1度だけ行ってからm.Run() を実行する。
// 戻り値はos.Exitに渡すための終了コード。
//
// SetupTx / GetTestDBのいずれもlazy initをサポートしているため、main_test.goの作成は任意。
// パッケージでeager initしたい場合のみTestMainで呼び出す。
//
// 使用例:
//
//	func TestMain(m *testing.M) {
//	    os.Exit(testutil.SetupTestMain(m))
//	}
func SetupTestMain(m *testing.M) int {
	initTestDB()
	return m.Run()
}

// SetupTxはテスト用のトランザクションをセットアップする。
// DB接続はsync.Onceで1回だけ確立し、パッケージ内の全テストで共有する。
// テスト終了時にはトランザクションのロールバックのみ実行する。
func SetupTx(t testing.TB) (*sql.DB, *sql.Tx) {
	t.Helper()

	return setupTx(t, nil)
}

// SetupTxRepeatableReadはSetupTxのトランザクションをREPEATABLE READに
// 固定したもの。他パッケージがコミットした行の安定したスナップショットをテストに
// 与えつつ、自身の後続の書き込みは引き続き見えるようにする。
//
// テスト対象のクエリが、そのテストが作った行にスコープされない場合 (テーブル全体を
// 走査する回復クエリや集計など) に使う。`go test ./...` はパッケージごとに別プロセス
// で同じテストDBを共有するため、他パッケージが実行中にコミットした行が同一テスト
// 内の2つのクエリの間で現れ、結果全体に対するアサーションが壊れる。
func SetupTxRepeatableRead(t testing.TB) (*sql.DB, *sql.Tx) {
	t.Helper()

	return setupTx(t, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
}

// setupTxは指定したオプションでテスト用トランザクションを開き、ロールバックを
// 登録する。
func setupTx(t testing.TB, opts *sql.TxOptions) (*sql.DB, *sql.Tx) {
	t.Helper()

	initTestDB()

	tx, err := testDB.BeginTx(context.Background(), opts)
	if err != nil {
		t.Fatalf("トランザクションの開始に失敗: %v", err)
	}

	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			t.Errorf("トランザクションのロールバックに失敗: %v", err)
		}
	})

	return testDB, tx
}

// GetTestDBは共有DB接続プールへの参照を返す。
// UseCaseが内部でdb.BeginTxを開く場合、テスト側でアウターTxを張るとUseCaseの内部Txから
// 前提データが見えなくなるため、Txで包まずにDBに直接コミットする必要がある。
// そのようなUseCaseテストで使用する。
func GetTestDB() *sql.DB {
	initTestDB()
	return testDB
}

// initTestDBはテスト用DB接続プールの初期化をsync.Onceにより1度だけ実行する。
// SetupTestMain / SetupTx / GetTestDBのいずれから呼ばれても同じ接続を共有する。
func initTestDB() {
	testDBOnce.Do(func() {
		// テスト用にbcryptコストを下げる (DefaultCost 10 → MinCost 4で約64倍高速化)
		auth.BcryptCost = auth.TestBcryptCost

		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			dsn = "postgres://postgres:postgres@postgresql:5432/mewst_test?sslmode=disable"
		}

		db, err := sql.Open("postgres", dsn)
		if err != nil {
			panic(fmt.Sprintf("テスト用データベースへの接続に失敗: %v", err))
		}

		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)

		if err := db.Ping(); err != nil {
			panic(fmt.Sprintf("テスト用データベースへのping失敗: %v", err))
		}

		testDB = db
	})
}

// MustParseUUIDは文字列をUUIDに変換する (パニックする可能性あり)
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}

// QueriesWithTxはトランザクションを使用する*query.Queriesを返す
// Repositoryテスト用のDIヘルパー
func QueriesWithTx(tx *sql.Tx) *query.Queries {
	return query.New(tx)
}

// AcquireMewstWebLockはmewst-webのoauth_applications行を保護する
// プロセス間のadvisory lockを獲得できるまでブロックし、t.Cleanupで解放する。
//
// `go test ./...` はパッケージごとに別プロセスで実行され、同じテストDBを
// 共有する。固定uidのmewst-web行をコミットする (または不在を前提とする)
// テストはパッケージをまたいで競合する: 同時INSERTはuidのUNIQUE
// インデックスで衝突し、一方のパッケージのcleanupが他方の依存中の行を削除
// しうる。コミットされたmewst-web行に触れるテストは、必ず先にこのロックを
// 獲得してパッケージ間で直列化すること。
func AcquireMewstWebLock(t testing.TB) {
	t.Helper()

	initTestDB()
	ctx := context.Background()

	// 専用コネクションを確保する。pg_advisory_lockはセッション単位のため、
	// 同じコネクション上で保持・解放する必要がある。
	conn, err := testDB.Conn(ctx)
	if err != nil {
		t.Fatalf("advisory lock用コネクションの取得に失敗: %v", err)
	}

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, mewstWebLockKey); err != nil {
		_ = conn.Close()
		t.Fatalf("advisory lockの獲得に失敗: %v", err)
	}

	t.Cleanup(func() {
		if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, mewstWebLockKey); err != nil {
			t.Errorf("advisory lockの解放に失敗: %v", err)
		}
		_ = conn.Close()
	})
}
