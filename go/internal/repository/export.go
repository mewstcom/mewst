package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// ErrActiveExportExistsは、プロフィールにqueuedまたはstartedの
// エクスポートが既にあるときにCreateが返す。判定するのはactiveなstatusに対する
// 部分ユニークインデックスであるため、この答えは、呼び出し側が事前に見て取れた
// ケースだけでなく、別transactionの同時Createもカバーする。
//
// ドライバーのエラーではなくsentinelにするのは、呼び出し側がDBドライバーを
// importしたり制約名で判定したりせずにこのケースを識別できるようにするため。
var ErrActiveExportExists = errors.New("プロフィールに進行中のエクスポートが既に存在する")

// exportsActiveProfileIndexは、プロフィールごとにqueued / startedの
// エクスポートを1件までとする部分ユニークインデックス。Createはドライバーの
// ユニーク制約違反をこの名前で照合し、別の制約 (後から追加されたものなど) の違反を
// 進行中のエクスポートとして報告しないようにする。
const exportsActiveProfileIndex = "index_exports_on_profile_id_where_active"

// ExportRepositoryはエクスポートのリポジトリ。
type ExportRepository struct {
	q *query.Queries
}

// NewExportRepositoryはExportRepositoryを生成する。
func NewExportRepository(q *query.Queries) *ExportRepository {
	return &ExportRepository{q: q}
}

// WithTxはトランザクションを設定したExportRepositoryを返す。
func (r *ExportRepository) WithTx(tx *sql.Tx) *ExportRepository {
	return &ExportRepository{q: r.q.WithTx(tx)}
}

// CreateExportInputはエクスポート作成の入力パラメータ。
type CreateExportInput struct {
	ProfileID model.ProfileID
	ActorID   model.ActorID
}

// Createはqueuedのエクスポートを挿入し、プロフィールの現在keptな投稿を
// 同じPostgreSQL文で固定化する。statement snapshotによって、後から元投稿が
// 物理削除されてもアーカイブの入力は変わらない。プロフィールにすでに進行中の
// エクスポートがあると、activeなstatusに対する部分ユニークインデックスが
// ここでErrActiveExportExistsとして現れるため、同一プロフィールへの同時Createが
// 2件目を作ることはない。また、永続的な削除マーカーを確認する前にプロフィール行を
// lockするため、作成はプロフィールcleanupが確立する境界と直列化される。境界を越えた
// プロフィールでは (nil, nil) を返す。マーカーが設定済みで文が何も挿入しないため、
// 呼び出し側はこれを失敗ではなく「エクスポートを開始できない」として扱う。
func (r *ExportRepository) Create(ctx context.Context, input CreateExportInput) (*model.Export, error) {
	row, err := r.q.CreateExport(ctx, query.CreateExportParams{
		ProfileID: uuid.UUID(input.ProfileID),
		ActorID:   uuid.UUID(input.ActorID),
	})
	if err != nil {
		// 行が返らないのは削除ゲートがINSERTを拒否した場合である
		// (CreateExportのprofile_gate CTEを参照)。失敗ではないため、他の
		// 「行が無い」ケースと同じ形で返し、エラーにはしない。
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if isExportActiveUniqueViolation(err) {
			return nil, ErrActiveExportExists
		}
		return nil, err
	}
	return r.toModel(query.Export(row)), nil
}

// isExportActiveUniqueViolationは、errがプロフィールごとに実行中の
// エクスポートを1件までとする部分インデックスによるユニーク制約違反かどうかを返す。
func isExportActiveUniqueViolation(err error) bool {
	var pgErr *pq.Error
	if !errors.As(err, &pgErr) {
		return false
	}

	// 23505はunique_violationのSQLSTATE。
	return pgErr.Code == "23505" && pgErr.Constraint == exportsActiveProfileIndex
}

// FindByIDは指定IDのエクスポートを返す。行が存在しない場合は (nil, nil) を
// 返す。生成ジョブは対象をこの方法で解決する。ジョブが持つのはエクスポートID
// だけで、status、楽観ロックのトークン、アーカイブの生成対象となるプロフィールと
// actorは行が持つため。
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

// FindLatestByProfileIDはstatusを問わずプロフィールの最新エクスポートを
// 返す。1件も無い場合は (nil, nil) を返す。
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

// FindLatestSucceededByProfileIDはプロフィールの最新のsucceeded
// エクスポートを返す。succeededが無い場合は (nil, nil) を返す。これが保持
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

// MarkStartedはエクスポートをstartedに遷移させ、attempt_countを増やして
// started_atを更新する。status IN ('queued', 'started') と期待するupdated_atを
// ガードにすることで、初回の試行とRiverのリトライを許可しつつ、古い試行が新しい
// 試行を上書きするのを防ぐ。
//
// 更新後の行を返す。ガードが一致しなかった場合は (nil, nil) を返し、呼び出し側は
// 遷移成功ではなく競合として扱う。行を返すのは、そのupdated_atが同じ試行を
// MarkSucceeded / MarkFailedで終わらせるために提示するトークンだからである。
// 別に読み直すと、その間にcommitされた遷移によって、呼び出し側が自分の保持しない
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

// MarkSucceededはstartedのエクスポートをsucceededに遷移させ、
// uploadしたobject keyの記録、送信待ち完了通知の作成、申請時の投稿snapshotの
// 破棄を同じ文で行う。行が更新されたかを返し、falseはstatusまたは期待する
// updated_atが一致しなかった競合を表す。
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

// MarkFailedはstartedのエクスポートをfailedに遷移させ、failedは終端状態の
// ため、同じ文で申請時の投稿snapshotを破棄する。行が更新されたかどうかを返す。
// falseはstatusまたは期待するupdated_atが一致しなかったことを意味し、呼び出し側は
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

// Requeueはタイムアウト回復のためstartedのエクスポートをqueuedに戻し、
// 行がqueuedの状態チェックを満たすようstarted_atをクリアしつつ、最大試行
// 判定のためattempt_countは保持する。行が更新されたかどうかを返す。falseは
// statusまたは期待するupdated_atが一致しなかったことを意味し、呼び出し側は
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

// ExportRecoveryCursorは時刻順の回復ページで最後に取得したエクスポートを
// 識別する。並び順のカラムは一覧メソッドごとに異なり (queuedはcreated_at、
// startedはstarted_at)、cursorは各メソッドが自身のカラムから組み立てる。
// 呼び出し側は自分で組み立てず、受け取ったcursorをそのまま渡し直す。誤った
// カラムから作ったcursorは、候補を静かに飛ばしたり重複して返したりする
// ため。
type ExportRecoveryCursor struct {
	Timestamp time.Time
	ID        model.ExportID
}

// exportRecoveryCursorParamsは任意のドメインcursorをsqlcパラメータへ
// 変換する。nilのcursorはゼロ時刻とゼロUUIDになり、保存されるどの行よりも前に
// 並ぶため、1ページ目に別のフラグを必要としない。
func exportRecoveryCursorParams(cursor *ExportRecoveryCursor) (time.Time, uuid.UUID) {
	if cursor == nil {
		return time.Time{}, uuid.Nil
	}
	return cursor.Timestamp, uuid.UUID(cursor.ID)
}

// nextExportRecoveryCursorはexportsの続きを取得するためのcursorを返す。
// ページが埋まらず走査が終端に達した場合はnilを返す。timestampOfは呼び出し元の
// 一覧メソッドの並び順カラムを最後の行から読む。そのカラムを持たない行はcursorに
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

// ListStaleQueuedはthresholdより前に作成されたqueuedのエクスポートを、
// cursorより後から古い順に1ページ返し、併せて次ページ用のcursorを返す。nilの
// cursorは最古の行から始め、次ページ用のcursorがnilなら走査は終端に達している。
// pageSizeは1以上である必要がある。0は空のページを返し、負値はクエリがエラーに
// なる。リコンシリエーションは一意な生成ジョブがすでに存在する場合もcursorを進め、
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

// ListStaleStartedは現在の試行がthresholdより前に始まったstartedの
// エクスポートを、cursorより後から古い順に1ページ返し、併せて次ページ用のcursor
// を返す。nilのcursorは最古の行から始め、次ページ用のcursorがnilなら走査は
// 終端に達している。pageSizeは1以上である必要がある。0は空のページを返し、負値は
// クエリがエラーになる。呼び出し側はattempt_countを見て再投入とfailedを判断し、
// すでに回復中の処理より後へcursorを進める。
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

// ListOldSucceededByProfileIDはプロフィールのsucceededのうち最新の1件を
// 除いたものを、古い順に1ページ返す。これらはR2オブジェクトと行を削除すべき
// cleanupの候補で、最新のsucceededは決して含まれないため、現在のダウンロード
// 対象は保護される。pageSizeは1以上である必要がある。0は空のページを返し、負値は
// クエリがエラーになる。cleanupは処理した行を削除するため、次回の実行では残りの
// 候補が先頭に現れる。よってcursorは不要。
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

// ListByProfileIDはプロフィールのエクスポートをstatusを問わず古い順に
// 1ページ返す。外部キーがON DELETE NO ACTIONであり、オブジェクトストレージが
// CASCADEの及ばない場所にあるため、プロフィールの削除は行を消す前にこの削除を
// 行う必要がある。それを支えるメソッド。pageSizeは1以上である必要がある。
// 0は空のページを返し、負値はクエリがエラーになる。呼び出し側が処理した行を
// 削除するため、次の呼び出しでは残りが先頭に現れる。よってcursorは不要。
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

// ListProfileIDsWithOldSucceededはsucceededのエクスポートを2件以上持つ
// プロフィールIDを、cursorより後から1ページ返し、併せて次ページ用のcursorを
// 返す。nilのcursorは最小のprofile IDから始め、次ページ用のcursorがnilなら
// 走査は終端に達している。pageSizeは1以上である必要がある。0は空のページを返し、
// 負値はクエリがエラーになる。リコンシリエーションは既存の一意なcleanupジョブより
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

// FindIDsRetainingObjectは与えたエクスポートIDのうち、オブジェクト
// ストレージ上のオブジェクトをまだ保持しているもの (failed以外のすべてのstatus)
// だけを返す。孤児回収が、どのR2オブジェクト (キーはエクスポートID) がまだ
// 保持されているかを判別するために使う。結果に無いIDが孤児の候補。入力が空の
// 場合はクエリを発行せずnilを返す。
//
// failedのエクスポートを保持側に含めないのは意図的である。オブジェクトを手放すのが
// 終端遷移であるため、failedの行の下に残ったオブジェクト (遷移と削除の間で終了した
// プロセス、あるいはストレージに触れずに停滞した試行を閉じたリコンシリエーションが
// 残したもの) こそ、孤児回収が回収すべきものである。
//
// 呼び出し側は一覧の全キーではなく、一定件数ずつのバッチで渡すことを前提とする。
// 孤児回収はexports/ 配下を全走査するが、ExportObjectStorage.ListPrefixが
// キーを1件ずつ渡すことで走査のメモリをO(1) に保っている。1回の呼び出しに
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

// DeleteはID指定でエクスポート行を1件削除する。cleanupはR2オブジェクト
// が消えた後にこれを呼ぶため、オブジェクトが残ったまま行が消えることはない。行が
// 削除されたかどうかを返す。
func (r *ExportRepository) Delete(ctx context.Context, id model.ExportID) (bool, error) {
	n, err := r.q.DeleteExport(ctx, uuid.UUID(id))
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// DeleteFailedByProfileIDはプロフィールのfailedなエクスポートを削除し、
// 消えた行数を返す。Createが新しいqueuedのエクスポートを挿入するtransactionで
// 呼ぶため、プロフィールが持つのは最新の成功1件と、進行中またはfailedの
// エクスポート1件までになる。boolではなく件数を返すのは、期待される件数が無い
// ため。プロフィールがfailedのエクスポートを1件も持たないことはあり、0件で
// あることは競合ではない。
func (r *ExportRepository) DeleteFailedByProfileID(ctx context.Context, profileID model.ProfileID) (int64, error) {
	return r.q.DeleteFailedExportsByProfileID(ctx, uuid.UUID(profileID))
}

// toModelはquery.Exportをmodel.Exportに変換する。
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

// toModelsはquery.Exportスライスをmodel.Exportスライスに変換する。
func (r *ExportRepository) toModels(rows []query.Export) []*model.Export {
	exports := make([]*model.Export, len(rows))
	for i, row := range rows {
		exports[i] = r.toModel(row)
	}
	return exports
}
