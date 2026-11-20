package viewmodel_test

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// TestNewExportはプロフィールのエクスポートが画面の状態と操作にどう変わるかを
// 固定する。最新のエクスポートと最新の成功したエクスポートが別の行になる場合も含む。
func TestNewExport(t *testing.T) {
	t.Parallel()

	succeeded := &model.Export{Status: model.ExportStatusSucceeded}

	tests := []struct {
		name            string
		latest          *model.Export
		latestSucceeded *model.Export
		available       bool
		wantState       viewmodel.ExportState
		wantCanStart    bool
		wantCanDownload bool
	}{
		{
			name:      "エクスポートが無ければ開始だけを出す",
			available: true,
			wantState: viewmodel.ExportStateNone, wantCanStart: true,
		},
		{
			name:      "queuedは進行中として開始を出さない",
			latest:    &model.Export{Status: model.ExportStatusQueued},
			available: true,
			wantState: viewmodel.ExportStateInProgress,
		},
		{
			name:      "startedは進行中として開始を出さない",
			latest:    &model.Export{Status: model.ExportStatusStarted},
			available: true,
			wantState: viewmodel.ExportStateInProgress,
		},
		{
			name:            "進行中でも以前の成功があればダウンロードを出す",
			latest:          &model.Export{Status: model.ExportStatusStarted},
			latestSucceeded: succeeded,
			available:       true,
			wantState:       viewmodel.ExportStateInProgress, wantCanDownload: true,
		},
		{
			name:            "成功では開始とダウンロードの両方を出す",
			latest:          succeeded,
			latestSucceeded: succeeded,
			available:       true,
			wantState:       viewmodel.ExportStateSucceeded, wantCanStart: true, wantCanDownload: true,
		},
		{
			name:      "失敗では再実行のために開始を出す",
			latest:    &model.Export{Status: model.ExportStatusFailed},
			available: true,
			wantState: viewmodel.ExportStateFailed, wantCanStart: true,
		},
		{
			name:            "失敗でも以前の成功があればダウンロードを出す",
			latest:          &model.Export{Status: model.ExportStatusFailed},
			latestSucceeded: succeeded,
			available:       true,
			wantState:       viewmodel.ExportStateFailed, wantCanStart: true, wantCanDownload: true,
		},
		{
			name:            "利用できないときは成功が残っていても操作を出さない",
			latest:          succeeded,
			latestSucceeded: succeeded,
			available:       false,
			wantState:       viewmodel.ExportStateUnavailable,
		},
		{
			// statusのCHECK制約によりこの状態には到達しない。このケースは
			// 現在のDBが作れる状態ではなく、将来statusが追加されたときに落ちる
			// フォールバックを固定する。
			name:      "未知のstatusは進行中として扱い開始を出さない",
			latest:    &model.Export{Status: model.ExportStatus("canceled")},
			available: true,
			wantState: viewmodel.ExportStateInProgress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := viewmodel.NewExport(viewmodel.ExportInput{
				Latest:          tt.latest,
				LatestSucceeded: tt.latestSucceeded,
				Available:       tt.available,
			})

			if got.State != tt.wantState {
				t.Errorf("State = %q、期待値 = %q", got.State, tt.wantState)
			}
			if got.CanStart != tt.wantCanStart {
				t.Errorf("CanStart = %v、期待値 = %v", got.CanStart, tt.wantCanStart)
			}
			if got.CanDownload != tt.wantCanDownload {
				t.Errorf("CanDownload = %v、期待値 = %v", got.CanDownload, tt.wantCanDownload)
			}
		})
	}
}
