// Package worker はバックグラウンドワーカー機能を提供します
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/email"
	mewstsentry "github.com/mewstcom/mewst/go/internal/sentry"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Client は River クライアントのラッパー
type Client struct {
	riverClient *river.Client[pgx.Tx]
	pool        *pgxpool.Pool
}

// NewClient creates a new River client. fanoutPostUC / addPostToTimelineUC
// depend on repository, so they cannot be built inside the worker package
// (worker is forbidden by depguard from importing repository / query); build
// them in main.go and inject them here. Dependencies that worker can import on
// its own (such as the email ones) are still built internally as before.
//
// [Ja] NewClient は新しい River クライアントを作成する。fanoutPostUC /
// addPostToTimelineUC は repository に依存するため worker 内では構築できず (worker は
// depguard で repository / query への依存が禁止)、main.go で構築して注入する。email 系の
// ように依存が worker から import 可能なものは従来どおり内部で構築する。
func NewClient(
	ctx context.Context,
	databaseURL string,
	cfg *config.Config,
	fanoutPostUC *usecase.FanoutPostUsecase,
	addPostToTimelineUC *usecase.AddPostToTimelineUsecase,
) (*Client, error) {
	// pgxpool の作成
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	// コネクションプール設定
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 5 * time.Minute
	poolConfig.MaxConnIdleTime = 2 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	// メール送信クライアントの作成
	var emailSender email.Sender
	if cfg.ResendAPIKey != "" && cfg.EmailFrom != "" {
		emailSender = email.NewResendSender(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
		slog.InfoContext(ctx, "Resend クライアントを初期化しました")
	} else {
		emailSender = email.NewNoopSender()
		slog.WarnContext(ctx, "Resend APIキーまたは送信元メールアドレスが設定されていないため、メール送信は無効です")
	}

	// River ワーカーの登録
	workers := river.NewWorkers()

	// メール送信ワーカーを登録
	confirmationSender := email.NewConfirmationSender(emailSender)
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(confirmationSender)
	river.AddWorker(workers, NewSendEmailConfirmationWorker(sendEmailConfirmationUC))
	slog.InfoContext(ctx, "SendEmailConfirmationWorker を登録しました")

	// タイムライン配信ワーカーを登録
	river.AddWorker(workers, NewFanoutPostWorker(fanoutPostUC))
	slog.InfoContext(ctx, "FanoutPostWorker を登録しました")
	river.AddWorker(workers, NewAddPostToTimelineWorker(addPostToTimelineUC))
	slog.InfoContext(ctx, "AddPostToTimelineWorker を登録しました")

	// River クライアントの作成
	// Middleware には Sentry エラーキャプチャ用の WorkerMiddleware を登録する。
	// これにより全 Worker のジョブ失敗が自動的に Sentry に送信される。
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		Logger:  slog.Default(),
		Middleware: []rivertype.Middleware{
			mewstsentry.RiverWorkerMiddleware(),
		},
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	return &Client{
		riverClient: riverClient,
		pool:        pool,
	}, nil
}

// Start は River クライアントを起動します
func (c *Client) Start(ctx context.Context) error {
	slog.InfoContext(ctx, "River クライアントを起動します")
	return c.riverClient.Start(ctx)
}

// Stop は River クライアントを停止します
func (c *Client) Stop(ctx context.Context) error {
	slog.InfoContext(ctx, "River クライアントを停止します")
	if err := c.riverClient.Stop(ctx); err != nil {
		return err
	}
	c.pool.Close()
	return nil
}

// Client は River クライアントへのアクセスを提供します
func (c *Client) Client() *river.Client[pgx.Tx] {
	return c.riverClient
}
