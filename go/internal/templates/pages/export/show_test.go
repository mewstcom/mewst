package export_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mewstcom/mewst/go/internal/i18n"
	exportpages "github.com/mewstcom/mewst/go/internal/templates/pages/export"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// TestShowTemplate_UnknownStateは、テンプレートが知らないExportStateが
// どの状態メッセージとダウンロードリンクのラベルへ落ちるかを固定する。現在
// viewmodel.NewExportはそのような値を返さないため、ハンドラーテストからこの分岐には
// 到達できず、テンプレートを直接レンダリングすることで固定する。
//
// フォールバックは進行中のメッセージで、viewmodel.exportStateが未知のstatusを
// 読むときと同じ倒し方になる。これにより2つの層は未知の状態を同じ形で説明し、
// どちらもエクスポートが1件も無いとは言わない。成功したエクスポートを引き続き
// ダウンロードできる場合、リンクは未知の最新状態が作ったとは言わず、以前の
// エクスポートと示す。
func TestShowTemplate_UnknownState(t *testing.T) {
	t.Parallel()

	ctx := i18n.SetLocale(t.Context(), "ja")

	var buf bytes.Buffer
	err := exportpages.Show(exportpages.ShowPageData{
		CSRFToken: "test-csrf-token",
		Export: viewmodel.Export{
			State:       viewmodel.ExportState("canceled"),
			CanDownload: true,
		},
	}).Render(ctx, &buf)
	if err != nil {
		t.Fatalf("テンプレートのレンダリングに失敗: %v", err)
	}

	body := buf.String()
	if want := "エクスポートを作成しています。完了したらメールでお知らせします。"; !strings.Contains(body, want) {
		t.Errorf("レスポンスに%qが含まれていません", want)
	}
	if unwant := "まだエクスポートを作成していません。"; strings.Contains(body, unwant) {
		t.Errorf("レスポンスに%qが含まれています", unwant)
	}
	if want := "以前のエクスポートをダウンロードする"; !strings.Contains(body, want) {
		t.Errorf("レスポンスに%qが含まれていません", want)
	}
}
