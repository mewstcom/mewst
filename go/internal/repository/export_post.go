package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ExportPostRepository is the repository for the post snapshot an export
// materializes at request time. ExportRepository.Create writes the snapshot in
// the same statement that creates the export, so this repository only reads it.
//
// [Ja] ExportPostRepository は export が申請時に固定化した投稿 snapshot の
// リポジトリ。snapshot の作成は export 行を作るのと同じ文で
// ExportRepository.Create が行うため、本リポジトリは読み取りだけを担う。
type ExportPostRepository struct {
	q *query.Queries
}

// NewExportPostRepository creates an ExportPostRepository.
//
// [Ja] NewExportPostRepository は ExportPostRepository を生成する。
func NewExportPostRepository(q *query.Queries) *ExportPostRepository {
	return &ExportPostRepository{q: q}
}

// WithTx returns a new ExportPostRepository bound to the transaction.
//
// [Ja] WithTx はトランザクションを設定した ExportPostRepository を返す。
func (r *ExportPostRepository) WithTx(tx *sql.Tx) *ExportPostRepository {
	return &ExportPostRepository{q: r.q.WithTx(tx)}
}

// PostMonth is one calendar month in an export's immutable post snapshot.
// LocalMonthStart names the month in Location, while StartsAt / EndsAt form a
// tight half-open UTC scan range containing that month's posts. The paging
// query applies the local month again, so a daylight saving fold at the month
// boundary cannot move a post between files.
//
// Location is the zone the month was computed in, and the paging listing reuses
// it rather than taking one of its own: a different zone would rebuild
// LocalMonthStart from a different wall clock and silently return a wrong page.
// A nil location is UTC, following the time package.
//
// [Ja] PostMonth は export の不変な投稿 snapshot に含まれる暦月 1 つ分。
// LocalMonthStart は Location における月を指し、StartsAt / EndsAt はその月の
// 投稿を含む狭い UTC 半開走査範囲を作る。分割取得クエリでもローカル月を
// 再適用するため、月境界の夏時間フォールドで投稿が別ファイルへ移動しない。
//
// Location はこの月を算出したゾーンで、分割取得は自前のゾーンを取らずにこれを
// 再利用する。別のゾーンを使うと LocalMonthStart を別の壁時計から組み立てる
// ことになり、静かに誤ったページを返すため。nil の location は time パッケージに
// 従い UTC として扱われる。
type PostMonth struct {
	LocalMonthStart time.Time
	StartsAt        time.Time
	EndsAt          time.Time
	PostCount       int64
	Location        *time.Location
}

// ListExportPostMonthsByExportIDInput is the input for listing the post months
// materialized for one export.
//
// [Ja] ListExportPostMonthsByExportIDInput は 1 つの export に固定化された投稿月
// 一覧を取得する入力パラメータ。
type ListExportPostMonthsByExportIDInput struct {
	ExportID model.ExportID
	Location *time.Location
}

// ListMonthsByExportID returns every calendar month in the export's immutable
// post snapshot, oldest first, with the post count and a UTC scan range
// containing that month's posts. It returns an empty slice when the snapshot
// has no post. Each returned month carries Location so the paging listing walks
// it in the same zone.
//
// Location must be a zone PostgreSQL can resolve as well, so callers pass a
// location obtained from time.LoadLocation (falling back to UTC when the
// user's time zone cannot be resolved) rather than a raw user-supplied string.
// A nil location is UTC, following the time package.
//
// [Ja] ListMonthsByExportID は export の不変な投稿 snapshot に含まれる暦月を
// すべて古い順に返し、併せて各月の投稿件数と、その投稿を含む UTC 走査範囲を
// 返す。投稿が 1 件も無い場合は空スライスを返す。返す各月は Location を持ち、
// 分割取得が同じゾーンで走査できるようにする。
//
// Location は PostgreSQL 側でも解決できるゾーンである必要があるため、呼び出し側は
// ユーザー入力の文字列をそのまま渡さず、time.LoadLocation で得た location
// (ユーザーのタイムゾーンを解決できない場合は UTC へフォールバック) を渡す。
// nil の location は time パッケージに従い UTC として扱われる。
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

// ExportPost is the request-time post data copied into an export snapshot. It
// intentionally contains only fields needed to render the archive and remains
// available even if Rails physically deletes the source post.
//
// [Ja] ExportPost は export snapshot へ申請時点で複製した投稿データ。アーカイブの
// 描画に必要なフィールドだけを持ち、Rails が元投稿を物理削除しても利用できる。
type ExportPost struct {
	ID          model.PostID
	Content     string
	PublishedAt time.Time
}

// PostCursor identifies the last post of a page in the (published_at, post_id)
// order the range listing uses. Callers pass back the cursor they received
// instead of assembling one, the same way the export recovery listings work.
//
// [Ja] PostCursor は範囲取得が使う (published_at, id) 順のページで最後に取得した
// 投稿を識別する。エクスポートの回復用一覧と同じく、呼び出し側は自分で組み立てず、
// 受け取った cursor をそのまま渡し直す。
type PostCursor struct {
	PublishedAt time.Time
	ID          model.PostID
}

// ListExportPostsByExportIDInRangeInput is the input for listing one page of a
// month from an export snapshot. Month is a row from ListMonthsByExportID,
// passed back whole the same way a cursor is: its calendar label, scan range
// and zone describe one month together, so a page built from parts of two
// months would silently return the wrong set.
//
// [Ja] ListExportPostsByExportIDInRangeInput は export snapshot の 1 か月分を
// 1 ページ取得する入力パラメータ。Month は ListMonthsByExportID が返した行で、
// cursor と同じくそのまま渡し直す。暦月のラベルと走査範囲とゾーンは 3 つで
// 1 か月を表すため、別々の月から組み合わせたページは静かに誤った集合を返す。
type ListExportPostsByExportIDInRangeInput struct {
	ExportID model.ExportID
	Month    PostMonth
	Cursor   *PostCursor
	PageSize int32
}

// ListByExportIDInRange returns a page from the export's immutable post
// snapshot that belongs to Month. Rows are oldest first and strictly after
// Cursor, along with the cursor for the next page. A nil cursor starts at the
// oldest post in the month, and a nil next cursor means the month has been
// walked to its end. PageSize must be at least 1: zero returns an empty page
// and a negative value makes the query fail.
//
// The month's bounds and cursor are instants. They are converted to UTC because
// export_posts stores these values in timestamp columns whose wall clocks are
// UTC. Month.LocalMonthStart is a calendar label rather than an instant and is
// sent without conversion.
//
// [Ja] ListByExportIDInRange は export の不変な投稿 snapshot のうち、Month の月に
// 属するものを Cursor より後から古い順に 1 ページ返す。併せて次ページ用の cursor
// を返す。nil の cursor はその月の最古の投稿から始め、次ページ用 cursor が nil
// なら月を終端まで走査している。PageSize は 1 以上である必要がある。
// 0 は空のページを返し、負値はクエリがエラーになる。
//
// 月の境界と cursor は時点を表す。export_posts はこれらを UTC の壁時計を持つ
// timestamp カラムへ保存しているため UTC へ変換する。Month.LocalMonthStart は
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

// exportLocationName returns a PostgreSQL-compatible zone name. A nil location
// follows the time package's convention and means UTC.
//
// [Ja] exportLocationName は PostgreSQL でも解決できるゾーン名を返す。nil の
// location は time パッケージの慣例に従い UTC を意味する。
func exportLocationName(location *time.Location) string {
	if location == nil {
		return time.UTC.String()
	}
	return location.String()
}

// postCursorParams converts an optional cursor to query parameters. A nil
// cursor becomes the zero timestamp and the zero UUID, which sort before every
// stored post, so the first page needs no separate flag.
//
// [Ja] postCursorParams は任意の cursor をクエリパラメータへ変換する。nil の
// cursor はゼロ時刻とゼロ UUID になり、保存されるどの投稿よりも前に並ぶため、
// 1 ページ目に別のフラグを必要としない。
func postCursorParams(cursor *PostCursor) (time.Time, uuid.UUID) {
	if cursor == nil {
		return time.Time{}, uuid.Nil
	}
	return cursor.PublishedAt.UTC(), uuid.UUID(cursor.ID)
}
