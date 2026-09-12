// Package testutil はテスト用のヘルパー関数とビルダーを提供する
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

// mewstWebLockKey is the advisory lock key guarding the mewst-web
// oauth_applications row (an arbitrary fixed value unique within this app).
//
// [Ja] mewstWebLockKey は mewst-web の oauth_applications 行を保護する
// advisory lock のキー (このアプリ内で一意な任意の固定値)。
const mewstWebLockKey int64 = 727577_001

var (
	testDB     *sql.DB
	testDBOnce sync.Once
)

// SetupTestMain はテストパッケージごとの TestMain で呼び出すヘルパー関数。
// bcrypt コストの低減と DB 接続プールの初期化をパッケージ内で 1 度だけ行ってから m.Run() を実行する。
// 戻り値は os.Exit に渡すための終了コード。
//
// SetupTx / GetTestDB のいずれも lazy init をサポートしているため、main_test.go の作成は任意。
// パッケージで eager init したい場合のみ TestMain で呼び出す。
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

// SetupTx はテスト用のトランザクションをセットアップする。
// DB 接続は sync.Once で 1 回だけ確立し、パッケージ内の全テストで共有する。
// テスト終了時にはトランザクションのロールバックのみ実行する。
func SetupTx(t testing.TB) (*sql.DB, *sql.Tx) {
	t.Helper()

	return setupTx(t, nil)
}

// SetupTxRepeatableRead is SetupTx with the transaction pinned to REPEATABLE
// READ, giving the test one stable view of the rows other packages committed
// while keeping its own later writes visible.
//
// Use it when the queries under test are not scoped to the rows the test
// created (a global recovery query, an aggregate over a whole table). `go test
// ./...` runs packages as separate processes sharing the same test DB, so rows
// another package commits mid-test would otherwise appear between two queries
// of the same test and break assertions about the full result.
//
// [Ja] SetupTxRepeatableRead は SetupTx のトランザクションを REPEATABLE READ に
// 固定したもの。他パッケージがコミットした行の安定したスナップショットをテストに
// 与えつつ、自身の後続の書き込みは引き続き見えるようにする。
//
// テスト対象のクエリが、そのテストが作った行にスコープされない場合 (テーブル全体を
// 走査する回復クエリや集計など) に使う。`go test ./...` はパッケージごとに別プロセス
// で同じテスト DB を共有するため、他パッケージが実行中にコミットした行が同一テスト
// 内の 2 つのクエリの間で現れ、結果全体に対するアサーションが壊れる。
func SetupTxRepeatableRead(t testing.TB) (*sql.DB, *sql.Tx) {
	t.Helper()

	return setupTx(t, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
}

// setupTx opens a test transaction with the given options and registers its
// rollback.
//
// [Ja] setupTx は指定したオプションでテスト用トランザクションを開き、ロールバックを
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

// GetTestDB は共有 DB 接続プールへの参照を返す。
// UseCase が内部で db.BeginTx を開く場合、テスト側でアウター Tx を張ると UseCase の内部 Tx から
// 前提データが見えなくなるため、Tx で包まずに DB に直接コミットする必要がある。
// そのような UseCase テストで使用する。
func GetTestDB() *sql.DB {
	initTestDB()
	return testDB
}

// initTestDB はテスト用 DB 接続プールの初期化を sync.Once により 1 度だけ実行する。
// SetupTestMain / SetupTx / GetTestDB のいずれから呼ばれても同じ接続を共有する。
func initTestDB() {
	testDBOnce.Do(func() {
		// テスト用にbcryptコストを下げる (DefaultCost 10 → MinCost 4 で約64倍高速化)
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

// MustParseUUID は文字列をUUIDに変換する (パニックする可能性あり)
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}

// QueriesWithTx はトランザクションを使用する*query.Queriesを返す
// Repository テスト用の DI ヘルパー
func QueriesWithTx(tx *sql.Tx) *query.Queries {
	return query.New(tx)
}

// AcquireMewstWebLock blocks until the cross-process advisory lock guarding
// the mewst-web oauth_applications row is acquired, and releases it via
// t.Cleanup.
//
// `go test ./...` runs packages as separate processes sharing the same test
// DB. Tests that commit (or assert the absence of) the fixed-uid mewst-web row
// race with each other across packages: concurrent inserts collide on the
// unique uid index, and one package's cleanup can delete the row while another
// package still depends on it. Every test that touches the committed mewst-web
// row must take this lock first so those tests serialize across packages.
//
// [Ja] AcquireMewstWebLock は mewst-web の oauth_applications 行を保護する
// プロセス間の advisory lock を獲得できるまでブロックし、t.Cleanup で解放する。
//
// `go test ./...` はパッケージごとに別プロセスで実行され、同じテスト DB を
// 共有する。固定 uid の mewst-web 行をコミットする (または不在を前提とする)
// テストはパッケージをまたいで競合する: 同時 INSERT は uid の UNIQUE
// インデックスで衝突し、一方のパッケージの cleanup が他方の依存中の行を削除
// しうる。コミットされた mewst-web 行に触れるテストは、必ず先にこのロックを
// 獲得してパッケージ間で直列化すること。
func AcquireMewstWebLock(t testing.TB) {
	t.Helper()

	initTestDB()
	ctx := context.Background()

	// Pin a dedicated connection: pg_advisory_lock is session-scoped, so the
	// lock must be held and released on the same connection.
	// [Ja] 専用コネクションを確保する。pg_advisory_lock はセッション単位のため、
	// 同じコネクション上で保持・解放する必要がある。
	conn, err := testDB.Conn(ctx)
	if err != nil {
		t.Fatalf("advisory lock 用コネクションの取得に失敗: %v", err)
	}

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, mewstWebLockKey); err != nil {
		_ = conn.Close()
		t.Fatalf("advisory lock の獲得に失敗: %v", err)
	}

	t.Cleanup(func() {
		if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, mewstWebLockKey); err != nil {
			t.Errorf("advisory lock の解放に失敗: %v", err)
		}
		_ = conn.Close()
	})
}
