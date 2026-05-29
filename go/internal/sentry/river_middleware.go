package sentry

import (
	"context"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// RiverWorkerMiddleware は river のジョブ実行をフックして、ジョブ内で発生したエラーを
// Sentry に自動送信するミドルウェアを返す。
//
// 動作:
//   - 各ジョブ実行ごとに独立した Hub を作る (Clone)。これにより並行実行されているジョブの
//     Scope が混ざらない。Clone 元は ctx に既に Hub が乗っていればそれを優先し、なければ
//     `sentry.CurrentHub()` を使う (river 経由の通常運用では ctx に Hub は乗っていない
//     ため後者が使われる)。
//   - Hub の Scope にジョブ種別 (`job.kind`) と試行回数 (`job.attempt`) をタグとしてセットする。
//     Sentry の UI 上で「どのジョブで起きたエラーか」「何回目の試行か」を絞り込めるようにする。
//   - 作成した Hub を ctx に bind してから `doInner(ctx)` を呼ぶ。
//     Worker 内部の `slog.ErrorContext` 経由のキャプチャ (sentryslog) も
//     このジョブ固有 Hub に紐付くため、タグが付いた状態で送信される。
//   - `doInner` のエラーが非 nil かつ無視対象 (context.Canceled 等) でなければ
//     `hub.CaptureException(err)` を呼んで「ジョブ全体としての失敗」を Sentry に送る。
//
// `worker.NewClient` 内の `river.Config.Middleware` に登録することで、すべての Worker に
// 自動適用される。新しい Worker を追加してもキャプチャ漏れが起きない。
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

// cloneHubForJob はジョブ固有の Hub を返す。ctx に既存の Hub があればそれを Clone し、
// なければ CurrentHub を Clone する。テストでは ctx に Hub を bind して注入できる。
func cloneHubForJob(ctx context.Context) *sentry.Hub {
	if existing := sentry.GetHubFromContext(ctx); existing != nil {
		return existing.Clone()
	}
	return sentry.CurrentHub().Clone()
}
