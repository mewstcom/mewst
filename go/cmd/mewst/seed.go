package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/database"
	"github.com/mewstcom/mewst/go/internal/seed"
)

// runSeed rebuilds the development database from the seed data.
//
// [Ja] runSeed は、開発用データベースをシードデータで作り直す。
func runSeed() {
	if err := seedDatabase(); err != nil {
		slog.Error("シードデータの投入に失敗しました", "error", err)
		os.Exit(1)
	}
}

// seedDatabase reaches the database and hands it to the seed.
//
// It carries no logic of its own beyond that: which environment the seed may
// run in, what it empties and what it creates are all decided in
// internal/seed, so that a second way of reaching the seed cannot arrive under
// different rules.
//
// The failure is returned rather than exited on, so that the connection is
// given back on the way out. A run that failed inside its transaction has left
// it to be rolled back, and os.Exit would drop the connection there instead.
//
// [Ja] seedDatabase はデータベースへ到達し、それをシードへ渡す。
//
// それ以外のロジックは持たない。どの環境で実行してよいか、何を空にし何を作るかは、
// いずれも internal/seed で決まる。シードへ辿り着く 2 つ目の経路が、異なる規則の
// もとで辿り着くことのないようにするため。
//
// 失敗をその場で終了させずに返すのは、抜けていく途中で接続を返すため。
// トランザクションの内側で失敗した実行はそのロールバックを控えており、os.Exit は
// そこで接続を落とすことになる。
func seedDatabase() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗: %w", err)
	}

	db, err := database.Connect(cfg.DatabaseDSN())
	if err != nil {
		return fmt.Errorf("データベース接続に失敗: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("データベース接続のクローズに失敗しました", "error", err)
		}
	}()

	return seed.NewRunner(db, os.Stderr).Run(context.Background())
}
