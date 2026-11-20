package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ExportPostRepositoryはexportが申請時に固定化した投稿snapshotの
// リポジトリ。snapshotの作成はexport行を作るのと同じ文で
// ExportRepository.Createが行うため、本リポジトリは読み取りだけを担う。
type ExportPostRepository struct {
	q *query.Queries
}

// NewExportPostRepositoryはExportPostRepositoryを生成する。
func NewExportPostRepository(q *query.Queries) *ExportPostRepository {
	return &ExportPostRepository{q: q}
}

// WithTxはトランザクションを設定したExportPostRepositoryを返す。
func (r *ExportPostRepository) WithTx(tx *sql.Tx) *ExportPostRepository {
	return &ExportPostRepository{q: r.q.WithTx(tx)}
}

// PostMonthはexportの不変な投稿snapshotに含まれる暦月1つ分。
// LocalMonthStartはLocationにおける月を指し、StartsAt / EndsAtはその月の
// 投稿を含む狭いUTC半開走査範囲を作る。分割取得クエリでもローカル月を
// 再適用するため、月境界の夏時間フォールドで投稿が別ファイルへ移動しない。
//
// Locationはこの月を算出したゾーンで、分割取得は自前のゾーンを取らずにこれを
// 再利用する。別のゾーンを使うとLocalMonthStartを別の壁時計から組み立てる
// ことになり、静かに誤ったページを返すため。nilのlocationはtimeパッケージに
// 従いUTCとして扱われる。
type PostMonth struct {
	LocalMonthStart time.Time
	StartsAt        time.Time
	EndsAt          time.Time
	PostCount       int64
	Location        *time.Location
}

// ListExportPostMonthsByExportIDInputは1つのexportに固定化された投稿月
// 一覧を取得する入力パラメータ。
type ListExportPostMonthsByExportIDInput struct {
	ExportID model.ExportID
	Location *time.Location
}

// ListMonthsByExportIDはexportの不変な投稿snapshotに含まれる暦月を
// すべて古い順に返し、併せて各月の投稿件数と、その投稿を含むUTC走査範囲を
// 返す。投稿が1件も無い場合は空スライスを返す。返す各月はLocationを持ち、
// 分割取得が同じゾーンで走査できるようにする。
//
// LocationはPostgreSQL側でも解決できるゾーンである必要があるため、呼び出し側は
// ユーザー入力の文字列をそのまま渡さず、time.LoadLocationで得たlocation
// (ユーザーのタイムゾーンを解決できない場合はUTCへフォールバック) を渡す。
// nilのlocationはtimeパッケージに従いUTCとして扱われる。
func (r *ExportPostRepository) ListMonthsByExportID(ctx context.Context, input ListExportPostMonthsByExportIDInput) ([]PostMonth, error) {
	rows, err := r.q.ListExportPostMonthsByExportID(ctx, query.ListExportPostMonthsByExportIDParams{
		TimeZone: exportLocationName(input.Location),
		ExportID: uuid.UUID(input.ExportID),
	})
	if err != nil {
		return nil, err
	}

	months := make([]PostMonth, len(rows))
	for i, row := range rows {
		months[i] = PostMonth{
			LocalMonthStart: row.LocalMonthStart,
			StartsAt:        row.StartsAt,
			EndsAt:          row.EndsAt,
			PostCount:       row.PostCount,
			Location:        input.Location,
		}
	}
	return months, nil
}

// ExportPostはexport snapshotへ申請時点で複製した投稿データ。アーカイブの
// 描画に必要なフィールドだけを持ち、Railsが元投稿を物理削除しても利用できる。
type ExportPost struct {
	ID          model.PostID
	Content     string
	PublishedAt time.Time
}

// PostCursorは範囲取得が使う (published_at, id) 順のページで最後に取得した
// 投稿を識別する。エクスポートの回復用一覧と同じく、呼び出し側は自分で組み立てず、
// 受け取ったcursorをそのまま渡し直す。
type PostCursor struct {
	PublishedAt time.Time
	ID          model.PostID
}

// ListExportPostsByExportIDInRangeInputはexport snapshotの1か月分を
// 1ページ取得する入力パラメータ。MonthはListMonthsByExportIDが返した行で、
// cursorと同じくそのまま渡し直す。暦月のラベルと走査範囲とゾーンは3つで
// 1か月を表すため、別々の月から組み合わせたページは静かに誤った集合を返す。
type ListExportPostsByExportIDInRangeInput struct {
	ExportID model.ExportID
	Month    PostMonth
	Cursor   *PostCursor
	PageSize int32
}

// ListByExportIDInRangeはexportの不変な投稿snapshotのうち、Monthの月に
// 属するものをCursorより後から古い順に1ページ返す。併せて次ページ用のcursor
// を返す。nilのcursorはその月の最古の投稿から始め、次ページ用cursorがnil
// なら月を終端まで走査している。PageSizeは1以上である必要がある。
// 0は空のページを返し、負値はクエリがエラーになる。
//
// 月の境界とcursorは時点を表す。export_postsはこれらをUTCの壁時計を持つ
// timestampカラムへ保存しているためUTCへ変換する。Month.LocalMonthStartは
// 時点ではなく暦月のラベルなので変換せずに渡す。
func (r *ExportPostRepository) ListByExportIDInRange(ctx context.Context, input ListExportPostsByExportIDInRangeInput) ([]*ExportPost, *PostCursor, error) {
	afterPublishedAt, afterID := postCursorParams(input.Cursor)
	rows, err := r.q.ListExportPostsByExportIDInRange(ctx, query.ListExportPostsByExportIDInRangeParams{
		ExportID:         uuid.UUID(input.ExportID),
		StartsAt:         input.Month.StartsAt.UTC(),
		EndsAt:           input.Month.EndsAt.UTC(),
		TimeZone:         exportLocationName(input.Month.Location),
		LocalMonthStart:  input.Month.LocalMonthStart,
		AfterPublishedAt: afterPublishedAt,
		AfterID:          afterID,
		PageSize:         input.PageSize,
	})
	if err != nil {
		return nil, nil, err
	}

	posts := make([]*ExportPost, len(rows))
	for i, row := range rows {
		posts[i] = &ExportPost{
			ID:          model.PostID(row.PostID),
			Content:     row.Content,
			PublishedAt: row.PublishedAt,
		}
	}

	var next *PostCursor
	if input.PageSize > 0 && len(posts) == int(input.PageSize) {
		last := posts[len(posts)-1]
		next = &PostCursor{PublishedAt: last.PublishedAt, ID: last.ID}
	}
	return posts, next, nil
}

// exportLocationNameはPostgreSQLでも解決できるゾーン名を返す。nilの
// locationはtimeパッケージの慣例に従いUTCを意味する。
func exportLocationName(location *time.Location) string {
	if location == nil {
		return time.UTC.String()
	}
	return location.String()
}

// postCursorParamsは任意のcursorをクエリパラメータへ変換する。nilの
// cursorはゼロ時刻とゼロUUIDになり、保存されるどの投稿よりも前に並ぶため、
// 1ページ目に別のフラグを必要としない。
func postCursorParams(cursor *PostCursor) (time.Time, uuid.UUID) {
	if cursor == nil {
		return time.Time{}, uuid.Nil
	}
	return cursor.PublishedAt.UTC(), uuid.UUID(cursor.ID)
}
