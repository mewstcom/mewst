package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ExportRepository is the repository for exports.
//
// [Ja] ExportRepository はエクスポートのリポジトリ。
type ExportRepository struct {
	q *query.Queries
}

// NewExportRepository creates an ExportRepository.
//
// [Ja] NewExportRepository は ExportRepository を生成する。
func NewExportRepository(q *query.Queries) *ExportRepository {
	return &ExportRepository{q: q}
}

// WithTx returns a new ExportRepository bound to the transaction.
//
// [Ja] WithTx はトランザクションを設定した ExportRepository を返す。
func (r *ExportRepository) WithTx(tx *sql.Tx) *ExportRepository {
	return &ExportRepository{q: r.q.WithTx(tx)}
}

// CreateExportInput is the input for creating an export.
//
// [Ja] CreateExportInput はエクスポート作成の入力パラメータ。
type CreateExportInput struct {
	ProfileID model.ProfileID
	ActorID   model.ActorID
}

// Create inserts a queued export and materializes the profile's currently kept
// posts in the same PostgreSQL statement. The statement snapshot fixes the
// archive's input even if a source post is physically deleted afterward. The
// partial unique index on active statuses surfaces as an error here when the
// profile already has an in-progress export, so a concurrent create for the
// same profile cannot produce a second one. The statement also locks the
// profile row before checking its persistent deletion marker, serializing
// creation with the boundary established by profile cleanup. A profile past
// that boundary yields (nil, nil): the marker is already set, so the statement
// inserts nothing and the caller has to report that no export can be started
// rather than treat it as a failure.
//
// [Ja] Create は queued のエクスポートを挿入し、プロフィールの現在 kept な投稿を
// 同じ PostgreSQL 文で固定化する。statement snapshot によって、後から元投稿が
// 物理削除されてもアーカイブの入力は変わらない。プロフィールにすでに進行中の
// エクスポートがあると、active な status に対する部分ユニークインデックスが
// ここでエラーとして現れるため、同一プロフィールへの同時 Create が 2 件目を
// 作ることはない。また、永続的な削除マーカーを確認する前にプロフィール行を lock
// するため、作成はプロフィール cleanup が確立する境界と直列化される。境界を越えた
// プロフィールでは (nil, nil) を返す。マーカーが設定済みで文が何も挿入しないため、
// 呼び出し側はこれを失敗ではなく「エクスポートを開始できない」として扱う。
func (r *ExportRepository) Create(ctx context.Context, input CreateExportInput) (*model.Export, error) {
	row, err := r.q.CreateExport(ctx, query.CreateExportParams{
		ProfileID: uuid.UUID(input.ProfileID),
		ActorID:   uuid.UUID(input.ActorID),
	})
	if err != nil {
		// No row back means the deletion gate rejected the insert; see the
		// profile_gate CTE in CreateExport. Nothing failed, so this is reported
		// the same way as any other absent row rather than as an error.
		//
		// [Ja] 行が返らないのは削除ゲートが INSERT を拒否した場合である
		// (CreateExport の profile_gate CTE を参照)。失敗ではないため、他の
		// 「行が無い」ケースと同じ形で返し、エラーにはしない。
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(query.Export(row)), nil
}

// FindByID returns the export with the given ID, or (nil, nil) when no row
// exists. The generation job resolves its target this way: the job carries only
// the export ID, and the row holds the status, the optimistic-lock token and
// the profile and actor the archive is built for.
//
// [Ja] FindByID は指定 ID のエクスポートを返す。行が存在しない場合は (nil, nil) を
// 返す。生成ジョブは対象をこの方法で解決する。ジョブが持つのはエクスポート ID
// だけで、status、楽観ロックのトークン、アーカイブの生成対象となるプロフィールと
// actor は行が持つため。
func (r *ExportRepository) FindByID(ctx context.Context, id model.ExportID) (*model.Export, error) {
	row, err := r.q.GetExportByID(ctx, uuid.UUID(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindLatestByProfileID returns the most recent export for the profile
// regardless of status, or (nil, nil) when the profile has none.
//
// [Ja] FindLatestByProfileID は status を問わずプロフィールの最新エクスポートを
// 返す。1 件も無い場合は (nil, nil) を返す。
func (r *ExportRepository) FindLatestByProfileID(ctx context.Context, profileID model.ProfileID) (*model.Export, error) {
	row, err := r.q.GetLatestExportByProfileID(ctx, uuid.UUID(profileID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// FindLatestSucceededByProfileID returns the most recent succeeded export for
// the profile, or (nil, nil) when the profile has no succeeded export. This is
// the single download target under the retention policy.
//
// [Ja] FindLatestSucceededByProfileID はプロフィールの最新の succeeded
// エクスポートを返す。succeeded が無い場合は (nil, nil) を返す。これが保持
// ポリシー上の唯一のダウンロード対象。
func (r *ExportRepository) FindLatestSucceededByProfileID(ctx context.Context, profileID model.ProfileID) (*model.Export, error) {
	row, err := r.q.GetLatestSucceededExportByProfileID(ctx, uuid.UUID(profileID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// MarkStarted moves an export into started, bumping attempt_count and stamping
// started_at. It guards on status IN ('queued', 'started') and the expected
// updated_at so both the first attempt and a River retry can proceed without a
// stale attempt overwriting a newer one.
//
// It returns the updated row, or (nil, nil) when a guard did not match, which
// the caller must treat as a conflict rather than a successful transition. The
// row is returned because its updated_at is the token the same attempt has to
// present to MarkSucceeded or MarkFailed; reading it back separately would let
// a transition committed in between hand the caller a token it does not own.
//
// [Ja] MarkStarted はエクスポートを started に遷移させ、attempt_count を増やして
// started_at を更新する。status IN ('queued', 'started') と期待する updated_at を
// ガードにすることで、初回の試行と River のリトライを許可しつつ、古い試行が新しい
// 試行を上書きするのを防ぐ。
//
// 更新後の行を返す。ガードが一致しなかった場合は (nil, nil) を返し、呼び出し側は
// 遷移成功ではなく競合として扱う。行を返すのは、その updated_at が同じ試行を
// MarkSucceeded / MarkFailed で終わらせるために提示するトークンだからである。
// 別に読み直すと、その間に commit された遷移によって、呼び出し側が自分の保持しない
// トークンを受け取り得る。
func (r *ExportRepository) MarkStarted(ctx context.Context, id model.ExportID, expectedUpdatedAt time.Time) (*model.Export, error) {
	row, err := r.q.MarkExportStarted(ctx, query.MarkExportStartedParams{
		ID:                uuid.UUID(id),
		ExpectedUpdatedAt: expectedUpdatedAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return r.toModel(row), nil
}

// MarkSucceeded moves a started export into succeeded, records the uploaded
// object's key, creates its pending completion notification, and discards its
// request-time post snapshot in the same statement. It reports whether a row
// was updated; false means the status or expected updated_at did not match and
// the caller must treat it as a conflict.
//
// [Ja] MarkSucceeded は started のエクスポートを succeeded に遷移させ、
// upload した object key の記録、送信待ち完了通知の作成、申請時の投稿 snapshot の
// 破棄を同じ文で行う。行が更新されたかを返し、false は status または期待する
// updated_at が一致しなかった競合を表す。
func (r *ExportRepository) MarkSucceeded(ctx context.Context, id model.ExportID, objectKey string, expectedUpdatedAt time.Time) (bool, error) {
	n, err := r.q.MarkExportSucceeded(ctx, query.MarkExportSucceededParams{
		ID:                uuid.UUID(id),
		ObjectKey:         sql.NullString{String: objectKey, Valid: true},
		ExpectedUpdatedAt: expectedUpdatedAt,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// MarkFailed moves a started export into failed and, since failed is terminal,
// discards the export's request-time post snapshot in the same statement. It
// reports whether a row was updated; false means the status or expected
// updated_at did not match and the caller must treat it as a conflict.
//
// [Ja] MarkFailed は started のエクスポートを failed に遷移させ、failed は終端状態の
// ため、同じ文で申請時の投稿 snapshot を破棄する。行が更新されたかどうかを返す。
// false は status または期待する updated_at が一致しなかったことを意味し、呼び出し側は
// 競合として扱う。
func (r *ExportRepository) MarkFailed(ctx context.Context, id model.ExportID, expectedUpdatedAt time.Time) (bool, error) {
	n, err := r.q.MarkExportFailed(ctx, query.MarkExportFailedParams{
		ID:                uuid.UUID(id),
		ExpectedUpdatedAt: expectedUpdatedAt,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Requeue moves a started export back to queued for timeout recovery, clearing
// started_at so the row satisfies the queued state check while keeping
// attempt_count for the max-attempt decision. It reports whether a row was
// updated; false means the status or expected updated_at did not match and the
// caller must treat it as a conflict.
//
// [Ja] Requeue はタイムアウト回復のため started のエクスポートを queued に戻し、
// 行が queued の状態チェックを満たすよう started_at をクリアしつつ、最大試行
// 判定のため attempt_count は保持する。行が更新されたかどうかを返す。false は
// status または期待する updated_at が一致しなかったことを意味し、呼び出し側は
// 競合として扱う。
func (r *ExportRepository) Requeue(ctx context.Context, id model.ExportID, expectedUpdatedAt time.Time) (bool, error) {
	n, err := r.q.RequeueExport(ctx, query.RequeueExportParams{
		ID:                uuid.UUID(id),
		ExpectedUpdatedAt: expectedUpdatedAt,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// ExportRecoveryCursor identifies the last export in a time-ordered recovery
// page. Each list method orders by a different column (created_at for queued
// and started_at for started) and builds the cursor from its own column.
// Callers pass back the cursor they received rather than assembling one;
// a cursor built from the wrong column would silently skip or repeat
// candidates.
//
// [Ja] ExportRecoveryCursor は時刻順の回復ページで最後に取得したエクスポートを
// 識別する。並び順のカラムは一覧メソッドごとに異なり (queued は created_at、
// started は started_at)、cursor は各メソッドが自身のカラムから組み立てる。
// 呼び出し側は自分で組み立てず、受け取った cursor をそのまま渡し直す。誤った
// カラムから作った cursor は、候補を静かに飛ばしたり重複して返したりする
// ため。
type ExportRecoveryCursor struct {
	Timestamp time.Time
	ID        model.ExportID
}

// exportRecoveryCursorParams converts an optional domain cursor to sqlc
// parameters. A nil cursor becomes the zero timestamp and the zero UUID, which
// sort before every stored row, so the first page needs no separate flag.
//
// [Ja] exportRecoveryCursorParams は任意のドメイン cursor を sqlc パラメータへ
// 変換する。nil の cursor はゼロ時刻とゼロ UUID になり、保存されるどの行よりも前に
// 並ぶため、1 ページ目に別のフラグを必要としない。
func exportRecoveryCursorParams(cursor *ExportRecoveryCursor) (time.Time, uuid.UUID) {
	if cursor == nil {
		return time.Time{}, uuid.Nil
	}
	return cursor.Timestamp, uuid.UUID(cursor.ID)
}

// nextExportRecoveryCursor returns the cursor that continues after exports, or
// nil when the page was not full and the walk has therefore reached the end.
// timestampOf reads the ordering column of the calling list method out of the
// last row; a row without that column cannot be paged past, so the walk ends
// there as well.
//
// [Ja] nextExportRecoveryCursor は exports の続きを取得するための cursor を返す。
// ページが埋まらず走査が終端に達した場合は nil を返す。timestampOf は呼び出し元の
// 一覧メソッドの並び順カラムを最後の行から読む。そのカラムを持たない行は cursor に
// できないため、その場合も走査を終える。
func nextExportRecoveryCursor(exports []*model.Export, pageSize int32, timestampOf func(*model.Export) *time.Time) *ExportRecoveryCursor {
	if pageSize <= 0 || len(exports) < int(pageSize) {
		return nil
	}

	last := exports[len(exports)-1]
	timestamp := timestampOf(last)
	if timestamp == nil {
		return nil
	}
	return &ExportRecoveryCursor{Timestamp: *timestamp, ID: last.ID}
}

// ListStaleQueued returns a page of queued exports created before the threshold,
// oldest first and strictly after cursor, along with the cursor for the next
// page. A nil cursor starts at the oldest row, and a nil next cursor means the
// walk reached the end. pageSize must be at least 1: zero returns an empty page
// and a negative value makes the query fail. Reconciliation advances the cursor
// even when a unique generation job already exists, so one stuck head cannot
// consume every run's new-work budget.
//
// Exports of a profile whose deletion has started are left out: generation
// stops at that profile's deletion marker, so re-enqueueing them would only
// produce jobs that return without touching the row. Profile deletion is what
// removes those rows.
//
// [Ja] ListStaleQueued は threshold より前に作成された queued のエクスポートを、
// cursor より後から古い順に 1 ページ返し、併せて次ページ用の cursor を返す。nil の
// cursor は最古の行から始め、次ページ用の cursor が nil なら走査は終端に達している。
// pageSize は 1 以上である必要がある。0 は空のページを返し、負値はクエリがエラーに
// なる。リコンシリエーションは一意な生成ジョブがすでに存在する場合も cursor を進め、
// 停滞した先頭候補が毎回の新規処理予算を占有しないようにする。
//
// 削除が始まったプロフィールのエクスポートは返さない。生成はそのプロフィールの削除
// マーカーで止まるため、再投入しても行に触れずに戻るジョブが生まれるだけである。
// これらの行を消すのは親削除である。
func (r *ExportRepository) ListStaleQueued(ctx context.Context, threshold time.Time, cursor *ExportRecoveryCursor, pageSize int32) ([]*model.Export, *ExportRecoveryCursor, error) {
	afterTime, afterID := exportRecoveryCursorParams(cursor)
	rows, err := r.q.ListStaleQueuedExports(ctx, query.ListStaleQueuedExportsParams{
		Threshold: threshold,
		AfterTime: afterTime,
		AfterID:   afterID,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, nil, err
	}

	exports := r.toModels(rows)
	next := nextExportRecoveryCursor(exports, pageSize, func(export *model.Export) *time.Time {
		return &export.CreatedAt
	})
	return exports, next, nil
}

// ListStaleStarted returns a page of started exports whose current attempt began
// before the threshold, oldest first and strictly after cursor, along with the
// cursor for the next page. A nil cursor starts at the oldest row, and a nil
// next cursor means the walk reached the end. pageSize must be at least 1: zero
// returns an empty page and a negative value makes the query fail. The caller
// decides between requeue and failed based on attempt_count and advances the
// cursor past work already being recovered.
//
// [Ja] ListStaleStarted は現在の試行が threshold より前に始まった started の
// エクスポートを、cursor より後から古い順に 1 ページ返し、併せて次ページ用の cursor
// を返す。nil の cursor は最古の行から始め、次ページ用の cursor が nil なら走査は
// 終端に達している。pageSize は 1 以上である必要がある。0 は空のページを返し、負値は
// クエリがエラーになる。呼び出し側は attempt_count を見て再投入と failed を判断し、
// すでに回復中の処理より後へ cursor を進める。
func (r *ExportRepository) ListStaleStarted(ctx context.Context, threshold time.Time, cursor *ExportRecoveryCursor, pageSize int32) ([]*model.Export, *ExportRecoveryCursor, error) {
	afterTime, afterID := exportRecoveryCursorParams(cursor)
	rows, err := r.q.ListStaleStartedExports(ctx, query.ListStaleStartedExportsParams{
		Threshold: sql.NullTime{Time: threshold, Valid: true},
		AfterTime: afterTime,
		AfterID:   afterID,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, nil, err
	}

	exports := r.toModels(rows)
	next := nextExportRecoveryCursor(exports, pageSize, func(export *model.Export) *time.Time {
		return export.StartedAt
	})
	return exports, next, nil
}

// ListOldSucceededByProfileID returns a page of the profile's succeeded exports
// except the most recent one, oldest first. These are the cleanup candidates
// whose R2 object and row should be deleted; the latest succeeded export is
// never included, so the current download target is protected. pageSize must be
// at least 1: zero returns an empty page and a negative value makes the query
// fail. No cursor is needed because cleanup deletes the rows it processed, so
// the next run sees the remaining candidates at the head.
//
// [Ja] ListOldSucceededByProfileID はプロフィールの succeeded のうち最新の 1 件を
// 除いたものを、古い順に 1 ページ返す。これらは R2 オブジェクトと行を削除すべき
// cleanup の候補で、最新の succeeded は決して含まれないため、現在のダウンロード
// 対象は保護される。pageSize は 1 以上である必要がある。0 は空のページを返し、負値は
// クエリがエラーになる。cleanup は処理した行を削除するため、次回の実行では残りの
// 候補が先頭に現れる。よって cursor は不要。
func (r *ExportRepository) ListOldSucceededByProfileID(ctx context.Context, profileID model.ProfileID, pageSize int32) ([]*model.Export, error) {
	rows, err := r.q.ListOldSucceededExportsByProfileID(ctx, query.ListOldSucceededExportsByProfileIDParams{
		ProfileID: uuid.UUID(profileID),
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListByProfileID returns a page of the profile's exports whatever their
// status, oldest first. It backs the deletion a profile removal has to perform
// before the row itself can go, since the foreign key is ON DELETE NO ACTION
// and the object storage is beyond what a cascade could reach. pageSize must be
// at least 1: zero returns an empty page and a negative value makes the query
// fail. No cursor is needed because the caller deletes the rows it processed,
// so the next call sees the remaining ones at the head.
//
// [Ja] ListByProfileID はプロフィールのエクスポートを status を問わず古い順に
// 1 ページ返す。外部キーが ON DELETE NO ACTION であり、オブジェクトストレージが
// CASCADE の及ばない場所にあるため、プロフィールの削除は行を消す前にこの削除を
// 行う必要がある。それを支えるメソッド。pageSize は 1 以上である必要がある。
// 0 は空のページを返し、負値はクエリがエラーになる。呼び出し側が処理した行を
// 削除するため、次の呼び出しでは残りが先頭に現れる。よって cursor は不要。
func (r *ExportRepository) ListByProfileID(ctx context.Context, profileID model.ProfileID, pageSize int32) ([]*model.Export, error) {
	rows, err := r.q.ListExportsByProfileID(ctx, query.ListExportsByProfileIDParams{
		ProfileID: uuid.UUID(profileID),
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	return r.toModels(rows), nil
}

// ListProfileIDsWithOldSucceeded returns a page of profile IDs that hold more
// than one succeeded export, ordered strictly after cursor, along with the
// cursor for the next page. A nil cursor starts at the smallest profile ID, and
// a nil next cursor means the walk reached the end. pageSize must be at least 1:
// zero returns an empty page and a negative value makes the query fail.
// Reconciliation advances past unique cleanup jobs that already exist before
// accepting more new work.
//
// [Ja] ListProfileIDsWithOldSucceeded は succeeded のエクスポートを 2 件以上持つ
// プロフィール ID を、cursor より後から 1 ページ返し、併せて次ページ用の cursor を
// 返す。nil の cursor は最小の profile ID から始め、次ページ用の cursor が nil なら
// 走査は終端に達している。pageSize は 1 以上である必要がある。0 は空のページを返し、
// 負値はクエリがエラーになる。リコンシリエーションは既存の一意な cleanup ジョブより
// 後へ進んでから、新しい処理を受理する。
func (r *ExportRepository) ListProfileIDsWithOldSucceeded(ctx context.Context, cursor *model.ProfileID, pageSize int32) ([]model.ProfileID, *model.ProfileID, error) {
	afterID := uuid.Nil
	if cursor != nil {
		afterID = uuid.UUID(*cursor)
	}

	ids, err := r.q.ListProfileIDsWithOldSucceededExports(ctx, query.ListProfileIDsWithOldSucceededExportsParams{
		AfterProfileID: afterID,
		PageSize:       pageSize,
	})
	if err != nil {
		return nil, nil, err
	}

	profileIDs := model.UUIDsToProfileIDs(ids)
	var next *model.ProfileID
	if pageSize > 0 && len(profileIDs) == int(pageSize) {
		last := profileIDs[len(profileIDs)-1]
		next = &last
	}
	return profileIDs, next, nil
}

// FindIDsRetainingObject returns the subset of the given export IDs whose
// export still retains an object in the object storage, which is every status
// but failed. Orphan recovery uses it to tell which R2 objects (keyed by export
// ID) are still claimed; the IDs absent from the result are orphan candidates.
// It returns nil for an empty input without querying.
//
// A failed export is deliberately not retaining: the terminal transition is
// what releases the object, so an object left under a failed row (a process
// that stopped between the transition and the deletion, or a reconciliation
// that closed a stale attempt without touching the storage) is exactly what
// orphan recovery has to collect.
//
// Callers are expected to pass one batch at a time rather than every key of a
// full listing. Orphan recovery walks the whole exports/ prefix, and
// ExportObjectStorage.ListPrefix hands keys over one by one so the walk stays
// O(1) in memory; accumulating every key just to make a single call here would
// give that back.
//
// [Ja] FindIDsRetainingObject は与えたエクスポート ID のうち、オブジェクト
// ストレージ上のオブジェクトをまだ保持しているもの (failed 以外のすべての status)
// だけを返す。孤児回収が、どの R2 オブジェクト (キーはエクスポート ID) がまだ
// 保持されているかを判別するために使う。結果に無い ID が孤児の候補。入力が空の
// 場合はクエリを発行せず nil を返す。
//
// failed のエクスポートを保持側に含めないのは意図的である。オブジェクトを手放すのが
// 終端遷移であるため、failed の行の下に残ったオブジェクト (遷移と削除の間で終了した
// プロセス、あるいはストレージに触れずに停滞した試行を閉じたリコンシリエーションが
// 残したもの) こそ、孤児回収が回収すべきものである。
//
// 呼び出し側は一覧の全キーではなく、一定件数ずつのバッチで渡すことを前提とする。
// 孤児回収は exports/ 配下を全走査するが、ExportObjectStorage.ListPrefix が
// キーを 1 件ずつ渡すことで走査のメモリを O(1) に保っている。1 回の呼び出しに
// するために全キーを貯めると、その利点を失う。
func (r *ExportRepository) FindIDsRetainingObject(ctx context.Context, ids []model.ExportID) ([]model.ExportID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	found, err := r.q.ListExportIDsRetainingObject(ctx, model.ExportIDsToUUIDs(ids))
	if err != nil {
		return nil, err
	}
	return model.UUIDsToExportIDs(found), nil
}

// Delete removes a single export row by ID. Cleanup calls it after the R2
// object is gone, so a row is never removed while its object still exists. It
// reports whether a row was deleted.
//
// [Ja] Delete は ID 指定でエクスポート行を 1 件削除する。cleanup は R2 オブジェクト
// が消えた後にこれを呼ぶため、オブジェクトが残ったまま行が消えることはない。行が
// 削除されたかどうかを返す。
func (r *ExportRepository) Delete(ctx context.Context, id model.ExportID) (bool, error) {
	n, err := r.q.DeleteExport(ctx, uuid.UUID(id))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// toModel converts a query.Export to a model.Export.
//
// [Ja] toModel は query.Export を model.Export に変換する。
func (r *ExportRepository) toModel(row query.Export) *model.Export {
	var objectKey *string
	if row.ObjectKey.Valid {
		objectKey = &row.ObjectKey.String
	}

	var startedAt *time.Time
	if row.StartedAt.Valid {
		startedAt = &row.StartedAt.Time
	}

	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		finishedAt = &row.FinishedAt.Time
	}

	return &model.Export{
		ID:           model.ExportID(row.ID),
		ProfileID:    model.ProfileID(row.ProfileID),
		ActorID:      model.ActorID(row.ActorID),
		Status:       model.ExportStatus(row.Status),
		ObjectKey:    objectKey,
		AttemptCount: row.AttemptCount,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

// toModels converts a slice of query.Export to a slice of model.Export.
//
// [Ja] toModels は query.Export スライスを model.Export スライスに変換する。
func (r *ExportRepository) toModels(rows []query.Export) []*model.Export {
	exports := make([]*model.Export, len(rows))
	for i, row := range rows {
		exports[i] = r.toModel(row)
	}
	return exports
}
