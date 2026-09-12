package viewmodel_test

import (
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// TestNewExport pins how the profile's exports become the page's state and
// actions, including the cases where the latest export and the latest
// succeeded one are different rows.
//
// [Ja] TestNewExport はプロフィールのエクスポートが画面の状態と操作にどう変わるかを
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
			name:      "queued は進行中として開始を出さない",
			latest:    &model.Export{Status: model.ExportStatusQueued},
			available: true,
			wantState: viewmodel.ExportStateInProgress,
		},
		{
			name:      "started は進行中として開始を出さない",
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
			// The status check constraint makes this unreachable, so the case
			// pins the fallback a future status would land in rather than a
			// state the database can produce today.
			//
			// [Ja] status の CHECK 制約によりこの状態には到達しない。このケースは
			// 現在の DB が作れる状態ではなく、将来 status が追加されたときに落ちる
			// フォールバックを固定する。
			name:      "未知の status は進行中として扱い開始を出さない",
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
				t.Errorf("State = %q, want %q", got.State, tt.wantState)
			}
			if got.CanStart != tt.wantCanStart {
				t.Errorf("CanStart = %v, want %v", got.CanStart, tt.wantCanStart)
			}
			if got.CanDownload != tt.wantCanDownload {
				t.Errorf("CanDownload = %v, want %v", got.CanDownload, tt.wantCanDownload)
			}
		})
	}
}
