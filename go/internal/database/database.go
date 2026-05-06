// Package database はデータベース接続を管理します
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Connect はPostgreSQLデータベースに接続します
// dsnはDATABASE_URL形式の接続文字列です
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("データベースへの接続に失敗しました: %w", err)
	}

	// 接続確認
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("データベースへのPingに失敗しました: %w", err)
	}

	return db, nil
}
