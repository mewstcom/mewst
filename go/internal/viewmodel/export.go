package viewmodel

import "github.com/mewstcom/mewst/go/internal/model"

// ExportState is which of the export page's mutually exclusive states to
// render. It is derived from the profile's latest export rather than taken
// from a single export's status, because the page also has states that no
// export row carries: having no export at all, and the feature being
// unavailable in this deployment.
//
// [Ja] ExportState はエクスポート画面が描画する排他的な状態。単一のエクスポートの
// status をそのまま使わずプロフィールの最新エクスポートから導出するのは、
// エクスポートが 1 件も無い状態と、そのデプロイで機能が利用できない状態という、
// どの export 行も持たない状態が画面にあるため。
type ExportState string

const (
	// ExportStateUnavailable means the deployment cannot run exports because
	// the object storage is not configured.
	//
	// [Ja] ExportStateUnavailable はオブジェクトストレージが未設定で、その
	// デプロイがエクスポートを実行できない状態。
	ExportStateUnavailable ExportState = "unavailable"

	// ExportStateNone means the profile has never requested an export.
	//
	// [Ja] ExportStateNone はプロフィールが一度もエクスポートを申請していない状態。
	ExportStateNone ExportState = "none"

	// ExportStateInProgress means the latest export is waiting for or running
	// its generation job.
	//
	// [Ja] ExportStateInProgress は最新のエクスポートが生成ジョブの開始を待っている
	// か、実行中の状態。
	ExportStateInProgress ExportState = "in_progress"

	// ExportStateSucceeded means the latest export finished and its zip is
	// downloadable.
	//
	// [Ja] ExportStateSucceeded は最新のエクスポートが完了し、その zip を
	// ダウンロードできる状態。
	ExportStateSucceeded ExportState = "succeeded"

	// ExportStateFailed means the latest export gave up. An older succeeded
	// export may still be downloadable.
	//
	// [Ja] ExportStateFailed は最新のエクスポートが諦めた状態。それより古い成功した
	// エクスポートはまだダウンロードできることがある。
	ExportStateFailed ExportState = "failed"
)

// Export is the view model for the export page. The page shows one state
// message and up to two actions, so the model carries the state plus whether
// each action applies, and the template only chooses among them.
//
// [Ja] Export はエクスポート画面の view model。画面は 1 つの状態メッセージと最大
// 2 つの操作を示すため、状態と各操作の可否を持ち、テンプレートはそこから選ぶだけに
// する。
type Export struct {
	// State is the state the page describes in text.
	//
	// [Ja] State は画面がテキストで説明する状態。
	State ExportState

	// CanStart reports whether the start button is shown. Starting is hidden
	// while an export is in progress, because the database allows only one
	// active export per profile.
	//
	// [Ja] CanStart は開始ボタンを表示するかどうか。DB がプロフィールごとに
	// 進行中のエクスポートを 1 件しか許さないため、進行中は開始を出さない。
	CanStart bool

	// CanDownload reports whether a succeeded export is available to download.
	// It stays true while a newer export is in progress or has failed, so the
	// previously produced zip remains reachable.
	//
	// [Ja] CanDownload はダウンロードできる成功したエクスポートがあるかどうか。
	// より新しいエクスポートが進行中または失敗のときも true のままとし、以前に
	// 作られた zip へ到達できるようにする。
	CanDownload bool
}

// ExportInput is what the export page's view model is derived from. The two
// exports are named fields rather than positional arguments because they share
// a type and carry different roles: the latest export decides the state, while
// the latest succeeded one decides whether a zip can be downloaded. Swapping
// them would still compile and would produce a plausible but wrong page.
//
// [Ja] ExportInput はエクスポート画面の view model を導出する元。2 つのエクスポートを
// 位置引数ではなく名前付きフィールドにするのは、型が同じで役割が異なるためである。
// 最新のエクスポートは状態を決め、最新の成功したエクスポートは zip をダウンロード
// できるかを決める。取り違えてもコンパイルは通り、もっともらしいが誤った画面になる。
type ExportInput struct {
	// Latest is the profile's most recent export regardless of status, or nil
	// when it has none.
	//
	// [Ja] Latest は status を問わないプロフィールの最新のエクスポート。1 件も
	// 無い場合は nil。
	Latest *model.Export

	// LatestSucceeded is the profile's most recent succeeded export, or nil
	// when no export has succeeded.
	//
	// [Ja] LatestSucceeded はプロフィールの最新の成功したエクスポート。成功した
	// エクスポートが無い場合は nil。
	LatestSucceeded *model.Export

	// Available reports whether the deployment can run exports at all.
	//
	// [Ja] Available はそのデプロイがそもそもエクスポートを実行できるかどうか。
	Available bool
}

// NewExport builds the export page's view model.
//
// When exports are unavailable, neither action is offered: both starting a new
// export and downloading an existing zip need the object storage that is
// missing, so offering them would only produce failures.
//
// [Ja] NewExport はエクスポート画面の view model を生成する。
//
// エクスポートが利用できない場合はどちらの操作も出さない。新しいエクスポートの開始も
// 既存 zip のダウンロードも、欠けているオブジェクトストレージを必要とするため、
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

// exportState maps the latest export to the state the page describes.
//
// An unrecognized status is treated as in progress. The status check constraint
// makes it unreachable today, and if a future status is added without updating
// this mapping, withholding the start button is the safer default: the page
// then shows no action whose effect on a row it cannot interpret.
//
// [Ja] exportState は最新のエクスポートを画面が説明する状態へ対応付ける。
//
// 認識できない status は進行中として扱う。status の CHECK 制約により現時点では
// 到達しないが、この対応付けを更新しないまま将来 status が追加された場合は、開始
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
