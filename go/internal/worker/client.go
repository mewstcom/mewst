// Package worker はバックグラウンドジョブ処理機能を提供します
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/email"
)

// Dependencies はWorkerクライアントの依存関係を保持する
type Dependencies struct {
	EmailSender email.Sender
}

// Client はジョブキューのクライアント
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClient は新しいWorkerクライアントを作成する
func NewClient(ctx context.Context, databaseURL string, deps Dependencies) (*Client, error) {
	// コネクションプール設定
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("データベースURL解析に失敗しました: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 5 * time.Minute
	poolConfig.MaxConnIdleTime = 2 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("コネクションプールの作成に失敗しました: %w", err)
	}

	// ワーカーの登録
	workers := river.NewWorkers()
	river.AddWorker(workers, NewSendEmailWorker(deps.EmailSender))

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("riverクライアントの作成に失敗しました: %w", err)
	}

	slog.Info("Workerクライアントを作成しました",
		"max_conns", poolConfig.MaxConns,
		"min_conns", poolConfig.MinConns,
	)

	return &Client{
		riverClient: riverClient,
		pool:        pool,
	}, nil
}

// Start はワーカーの処理を開始する
func (c *Client) Start(ctx context.Context) error {
	if err := c.riverClient.Start(ctx); err != nil {
		return fmt.Errorf("ワーカーの開始に失敗しました: %w", err)
	}
	return nil
}

// Stop はワーカーの処理を停止する
func (c *Client) Stop(ctx context.Context) error {
	if err := c.riverClient.Stop(ctx); err != nil {
		return fmt.Errorf("ワーカーの停止に失敗しました: %w", err)
	}
	c.pool.Close()
	return nil
}

// Insert はジョブをキューに追加する
func (c *Client) Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	return c.riverClient.Insert(ctx, args, nil)
}

// Client は内部のRiverクライアントへのアクセスを提供する（定期ジョブ用）
func (c *Client) Client() *river.Client[pgx.Tx] {
	return c.riverClient
}
