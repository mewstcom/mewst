package sentry

import (
	"context"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// RiverWorkerMiddlewareはriverのジョブ実行をフックして、ジョブ内で発生したエラーを
// Sentryに自動送信するミドルウェアを返す。
//
// 動作:
//   - 各ジョブ実行ごとに独立したHubを作る (Clone)。これにより並行実行されているジョブの
//     Scopeが混ざらない。Clone元はctxに既にHubが乗っていればそれを優先し、なければ
//     `sentry.CurrentHub()` を使う (river経由の通常運用ではctxにHubは乗っていない
//     ため後者が使われる)。
//   - HubのScopeにジョブ種別 (`job.kind`) と試行回数 (`job.attempt`) をタグとしてセットする。
//     SentryのUI上で「どのジョブで起きたエラーか」「何回目の試行か」を絞り込めるようにする。
//   - 作成したHubをctxにbindしてから `doInner(ctx)` を呼ぶ。
//     Worker内部の `slog.ErrorContext` 経由のキャプチャ (sentryslog) も
//     このジョブ固有Hubに紐付くため、タグが付いた状態で送信される。
//   - `doInner` のエラーが非nilかつ無視対象 (context.Canceled等) でなければ
//     `hub.CaptureException(err)` を呼んで「ジョブ全体としての失敗」をSentryに送る。
//
// `worker.NewClient` 内の `river.Config.Middleware` に登録することで、すべてのWorkerに
// 自動適用される。新しいWorkerを追加してもキャプチャ漏れが起きない。
func RiverWorkerMiddleware() rivertype.Middleware {
	return river.WorkerMiddlewareFunc(func(ctx context.Context, job *rivertype.JobRow, doInner func(ctx context.Context) error) error {
		hub := cloneHubForJob(ctx)
		hub.Scope().SetTag("job.kind", job.Kind)
		hub.Scope().SetTag("job.attempt", strconv.Itoa(job.Attempt))
		ctx = sentry.SetHubOnContext(ctx, hub)

		err := doInner(ctx)
		if err != nil && !shouldDropError(err) {
			hub.CaptureException(err)
		}
		return err
	})
}

// cloneHubForJobはジョブ固有のHubを返す。ctxに既存のHubがあればそれをCloneし、
// なければCurrentHubをCloneする。テストではctxにHubをbindして注入できる。
func cloneHubForJob(ctx context.Context) *sentry.Hub {
	if existing := sentry.GetHubFromContext(ctx); existing != nil {
		return existing.Clone()
	}
	return sentry.CurrentHub().Clone()
}
