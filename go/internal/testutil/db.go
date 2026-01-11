// Package testutil はテスト用のヘルパー関数とビルダーを提供する
package testutil

import (
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// SetupTestDB はテスト用のデータベース接続とトランザクションをセットアップする
// テスト終了時にトランザクションをロールバックしてクリーンアップする
func SetupTestDB(t *testing.T) (*sql.DB, *sql.Tx) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@postgresql:5432/mewst_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("データベースへの接続に失敗: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("データベースへの接続確認に失敗: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("トランザクションの開始に失敗: %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
		_ = db.Close()
	})

	return db, tx
}

// MustParseUUID は文字列をUUIDに変換する（パニックする可能性あり）
func MustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
