package viewmodel

import "github.com/mewstcom/mewst/go/internal/model"

// ExportStateはエクスポート画面が描画する排他的な状態。単一のエクスポートの
// statusをそのまま使わずプロフィールの最新エクスポートから導出するのは、
// エクスポートが1件も無い状態と、そのデプロイで機能が利用できない状態という、
// どのexport行も持たない状態が画面にあるため。
type ExportState string

const (
	// ExportStateUnavailableはオブジェクトストレージが未設定で、その
	// デプロイがエクスポートを実行できない状態。
	ExportStateUnavailable ExportState = "unavailable"

	// ExportStateNoneはプロフィールが一度もエクスポートを申請していない状態。
	ExportStateNone ExportState = "none"

	// ExportStateInProgressは最新のエクスポートが生成ジョブの開始を待っている
	// か、実行中の状態。
	ExportStateInProgress ExportState = "in_progress"

	// ExportStateSucceededは最新のエクスポートが完了し、そのzipを
	// ダウンロードできる状態。
	ExportStateSucceeded ExportState = "succeeded"

	// ExportStateFailedは最新のエクスポートが諦めた状態。それより古い成功した
	// エクスポートはまだダウンロードできることがある。
	ExportStateFailed ExportState = "failed"
)

// Exportはエクスポート画面のview model。画面は1つの状態メッセージと最大
// 2つの操作を示すため、状態と各操作の可否を持ち、テンプレートはそこから選ぶだけに
// する。
type Export struct {
	// Stateは画面がテキストで説明する状態。
	State ExportState

	// CanStartは開始ボタンを表示するかどうか。DBがプロフィールごとに
	// 進行中のエクスポートを1件しか許さないため、進行中は開始を出さない。
	CanStart bool

	// CanDownloadはダウンロードできる成功したエクスポートがあるかどうか。
	// より新しいエクスポートが進行中または失敗のときもtrueのままとし、以前に
	// 作られたzipへ到達できるようにする。
	CanDownload bool
}

// ExportInputはエクスポート画面のview modelを導出する元。2つのエクスポートを
// 位置引数ではなく名前付きフィールドにするのは、型が同じで役割が異なるためである。
// 最新のエクスポートは状態を決め、最新の成功したエクスポートはzipをダウンロード
// できるかを決める。取り違えてもコンパイルは通り、もっともらしいが誤った画面になる。
type ExportInput struct {
	// Latestはstatusを問わないプロフィールの最新のエクスポート。1件も
	// 無い場合はnil。
	Latest *model.Export

	// LatestSucceededはプロフィールの最新の成功したエクスポート。成功した
	// エクスポートが無い場合はnil。
	LatestSucceeded *model.Export

	// Availableはそのデプロイがそもそもエクスポートを実行できるかどうか。
	Available bool
}

// NewExportはエクスポート画面のview modelを生成する。
//
// エクスポートが利用できない場合はどちらの操作も出さない。新しいエクスポートの開始も
// 既存zipのダウンロードも、欠けているオブジェクトストレージを必要とするため、
// 出しても失敗を生むだけである。
func NewExport(input ExportInput) Export {
	if !input.Available {
		return Export{State: ExportStateUnavailable}
	}

	state := exportState(input.Latest)

	return Export{
		State:       state,
		CanStart:    state != ExportStateInProgress,
		CanDownload: input.LatestSucceeded != nil,
	}
}

// exportStateは最新のエクスポートを画面が説明する状態へ対応付ける。
//
// 認識できないstatusは進行中として扱う。statusのCHECK制約により現時点では
// 到達しないが、この対応付けを更新しないまま将来statusが追加された場合は、開始
// ボタンを出さない方が安全な既定となる。解釈できない行への影響が分からない操作を
// 画面に出さずに済むため。
func exportState(latest *model.Export) ExportState {
	if latest == nil {
		return ExportStateNone
	}

	switch latest.Status {
	case model.ExportStatusSucceeded:
		return ExportStateSucceeded
	case model.ExportStatusFailed:
		return ExportStateFailed
	default:
		return ExportStateInProgress
	}
}
