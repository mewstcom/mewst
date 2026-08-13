package usecase

import "time"

const (
	// ExportPostPageSize is how many posts the generation flow reads from an
	// export's snapshot at a time. Reading a whole month, let alone a whole
	// export, would make the memory the generation holds grow with the number of
	// posts, so the flow keeps only one page and streams it into the archive.
	//
	// The size is the point where fewer round trips stop paying off: walking a
	// 100,000 post snapshot takes about three times as long at 100 posts per
	// page, while pages ten times larger save almost nothing and hold ten times
	// as many posts in memory.
	//
	// [Ja] ExportPostPageSize は生成処理が export の snapshot から一度に読み取る
	// 投稿の件数。月全体、まして export 全体を読むと生成処理が保持するメモリが投稿
	// 件数に比例して増えるため、1 ページだけを保持してアーカイブへ流し込む。
	//
	// この件数は、ラウンドトリップを減らす効果が頭打ちになる地点。10 万件の
	// snapshot の走査は 1 ページ 100 件では約 3 倍の時間がかかる一方、10 倍大きな
	// ページにしてもほとんど短縮されず、メモリに載る投稿だけが 10 倍になる。
	ExportPostPageSize int32 = 1000

	// GenerateExportTimeout bounds one generation attempt. The bound exists so
	// that a stalled attempt (a hung upload, a lost connection) releases the
	// single-worker export queue and is recovered as a stale started export,
	// rather than blocking every other profile's export for as long as the
	// process lives.
	//
	// It is set far above the measured cost of the export size the requirements
	// are stated in, so that a slower database, a slower link to the object
	// storage, and profiles well beyond that size all finish inside it.
	//
	// [Ja] GenerateExportTimeout は生成 1 回の試行の上限。停止した試行 (固まった
	// アップロード、切れた接続) が、プロセスが生きている限り他プロフィールの
	// エクスポートを塞ぎ続けるのではなく、単一 worker の export キューを解放して
	// stale な started として回復されるようにするための上限。
	//
	// 値は要件が前提とするエクスポート規模の実測コストより十分大きく取る。DB が
	// 遅い場合、オブジェクトストレージへの回線が細い場合、その規模を大きく超える
	// プロフィールのいずれもこの時間内に完了するようにするため。
	GenerateExportTimeout = 15 * time.Minute
)
