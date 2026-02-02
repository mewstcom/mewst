// Package worker はバックグラウンドジョブ処理機能を提供します
package worker

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivertype"
)

// Client はジョブキューのクライアント
type Client struct {
	riverClient *river.Client[*sql.Tx]
}

// NewClient は新しいWorkerクライアントを作成する
func NewClient(ctx context.Context, db *sql.DB, workers *river.Workers) (*Client, error) {
	riverClient, err := river.NewClient(riverdatabasesql.New(db), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("riverクライアントの作成に失敗しました: %w", err)
	}

	return &Client{
		riverClient: riverClient,
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
	return nil
}

// Insert はジョブをキューに追加する
func (c *Client) Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	return c.riverClient.Insert(ctx, args, nil)
}
