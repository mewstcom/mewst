// Package testutil はテスト用のヘルパー関数とビルダーを提供する
package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/mewstcom/mewst/go/internal/auth"
)

var (
	testDB     *sql.DB
	testDBOnce sync.Once
)

// SetupTestDB はテスト用のデータベース接続とトランザクションをセットアップする
// DB接続はsync.Onceで1回だけ確立し、パッケージ内の全テストで共有する
// 各テスト終了時にはトランザクションのロールバックのみ実行する
func SetupTestDB(t *testing.T) (*sql.DB, *sql.Tx) {
	t.Helper()

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

	tx, err := testDB.Begin()
	if err != nil {
		t.Fatalf("トランザクションの開始に失敗: %v", err)
	}

	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			t.Errorf("トランザクションのロールバックに失敗: %v", err)
		}
	})

	return testDB, tx
}

// MustParseUUID は文字列をUUIDに変換する（パニックする可能性あり）
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
