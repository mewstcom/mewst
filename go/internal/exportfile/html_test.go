package exportfile

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/usecase"
)

// errWriteStopped is the failure the fixture writer returns once it stops
// accepting writes.
//
// [Ja] errWriteStopped は fixture の writer が書き込みの受け付けをやめた後に
// 返す失敗。
var errWriteStopped = errors.New("書き込みに失敗")

// stopAfterWriter accepts a fixed number of writes and fails every write after
// them, recording how many times it was called.
//
// [Ja] stopAfterWriter は決めた回数だけ書き込みを受け付け、それ以降の書き込みは
// すべて失敗させる。呼び出された回数を記録する。
type stopAfterWriter struct {
	accepts int
	writes  int
}

func (s *stopAfterWriter) Write(p []byte) (int, error) {
	s.writes++
	if s.accepts <= 0 {
		return 0, errWriteStopped
	}
	s.accepts--
	return len(p), nil
}

func TestWriteIndexHTML_SkipsFragmentsAfterFirstWriteError(t *testing.T) {
	t.Parallel()

	archive := usecase.ExportArchive{
		Locale: "ja",
		Months: []usecase.ExportArchiveMonth{
			{LocalMonthStart: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC), PostCount: 1},
		},
		GeneratedAt: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
	}

	// The document is written as more than two fragments, so a writer that
	// fails on its second write shows that the remaining fragments are skipped
	// and the first error is the one returned.
	//
	// [Ja] ドキュメントは 3 つ以上のフラグメントとして書き出されるため、2 回目の
	// 書き込みで失敗する writer を使えば、残りのフラグメントがスキップされ、
	// 最初のエラーがそのまま返ることを確認できる。
	w := &stopAfterWriter{accepts: 1}
	if err := writeIndexHTML(context.Background(), w, archive); !errors.Is(err, errWriteStopped) {
		t.Fatalf("writeIndexHTML のエラー = %v, want %v", err, errWriteStopped)
	}
	if w.writes != 2 {
		t.Errorf("書き込み回数 = %d, want 2 (最初の失敗以降はスキップされる)", w.writes)
	}
}
