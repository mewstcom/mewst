// Package testutil はテスト用のヘルパー関数とビルダーを提供する
package testutil

import (
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
func SetupTx(t *testing.T) (*sql.DB, *sql.Tx) {
	t.Helper()

	initTestDB()

	tx, err := testDB.Begin()
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
		// テスト用にbcryptコストを下げる（DefaultCost 10 → MinCost 4 で約64倍高速化）
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

// MustParseUUID は文字列をUUIDに変換する（パニックする可能性あり）
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
