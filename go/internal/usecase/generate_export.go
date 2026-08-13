package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// exportTerminalCleanupTimeout bounds the work done after the last attempt
// failed. That work runs on a context detached from the job's, which may
// already be canceled or timed out, so this timeout is what keeps the cleanup
// from hanging the worker after the attempt itself is over.
//
// [Ja] exportTerminalCleanupTimeout は最終試行が失敗した後の後処理の上限時間。
// 後処理はジョブの context (すでにキャンセル済みまたは timeout 済みのことがある)
// から切り離した context で実行するため、この timeout が、試行が終わった後の
// worker を後処理でハングさせないための歯止めになる。
const exportTerminalCleanupTimeout = 30 * time.Second

// errExportUploadStopped closes the archive side of the pipe when the upload
// stops reading. It only unblocks a writer that would otherwise wait forever
// for a reader that is gone, so upload drops it from what it reports rather
// than letting it stand in front of the failure that actually stopped the
// upload.
//
// [Ja] errExportUploadStopped は upload が読み取りを止めたときにアーカイブ側の
// pipe を閉じる。いなくなった読み手を永久に待ち続ける writer を解放するためだけの
// ものなので、upload は報告するエラーから取り除く。実際にアップロードを止めた失敗の
// 前に、これを並べないようにするため。
var errExportUploadStopped = errors.New("アップロードが終了したためアーカイブの書き出しを中断した")

// GenerateExportUsecase builds one export's zip and streams it to the object
// storage. It owns the whole attempt: it takes the export from queued (or from
// the started row a retry re-enters) through the upload to succeeded, and on
// the final attempt it closes the export as failed instead of leaving it
// started forever.
//
// Every state transition is guarded by the token the previous one produced, so
// an attempt that lost the row (to reconciliation, or to a retry that overtook
// it) fails its transition instead of overwriting the newer state.
//
// [Ja] GenerateExportUsecase は 1 件のエクスポートの zip を構築し、オブジェクト
// ストレージへストリーミングする。1 回の試行全体を担い、エクスポートを queued
// (リトライが再入する場合は started の行) からアップロードを経て succeeded まで
// 進める。最終試行では、started のまま放置せず failed としてエクスポートを閉じる。
//
// 各状態遷移は直前の遷移が生成したトークンでガードされるため、行を失った試行
// (リコンシリエーションや追い越したリトライによる) は、新しい状態を上書きせずに
// 遷移が失敗する。
type GenerateExportUsecase struct {
	exportRepo     *repository.ExportRepository
	exportPostRepo *repository.ExportPostRepository
	actorRepo      *repository.ActorRepository
	userRepo       *repository.UserRepository
	deletionGuard  ExportProfileDeletionGuard
	archiveBuilder ExportArchiveBuilder
	objectStorage  ExportObjectStorage
	dispatcher     *dispatcher.Dispatcher
}

// NewGenerateExportUsecase creates a GenerateExportUsecase.
//
// [Ja] NewGenerateExportUsecase は GenerateExportUsecase を生成する。
func NewGenerateExportUsecase(
	exportRepo *repository.ExportRepository,
	exportPostRepo *repository.ExportPostRepository,
	actorRepo *repository.ActorRepository,
	userRepo *repository.UserRepository,
	deletionGuard ExportProfileDeletionGuard,
	archiveBuilder ExportArchiveBuilder,
	objectStorage ExportObjectStorage,
	d *dispatcher.Dispatcher,
) *GenerateExportUsecase {
	return &GenerateExportUsecase{
		exportRepo:     exportRepo,
		exportPostRepo: exportPostRepo,
		actorRepo:      actorRepo,
		userRepo:       userRepo,
		deletionGuard:  deletionGuard,
		archiveBuilder: archiveBuilder,
		objectStorage:  objectStorage,
		dispatcher:     d,
	}
}

// GenerateExportInput is the input for generating one export.
//
// [Ja] GenerateExportInput は 1 件のエクスポートを生成する入力パラメータ。
type GenerateExportInput struct {
	// ExportID is the export to generate.
	//
	// [Ja] ExportID は生成対象のエクスポート。
	ExportID model.ExportID

	// IsFinalAttempt tells whether a failure of this attempt ends the export.
	// The job queue owns the retry budget, so the caller reports the verdict
	// rather than this use case rederiving it: on the final attempt the export
	// is closed as failed and its object is removed, and before it a failure
	// simply leaves the row started for the next attempt.
	//
	// [Ja] IsFinalAttempt はこの試行の失敗でエクスポートが終わるかどうかを表す。
	// 再試行の予算はジョブキューが持つため、本 UseCase が導出し直すのではなく
	// 呼び出し側が判定を渡す。最終試行ではエクスポートを failed として閉じ、その
	// オブジェクトを削除する。それ以前の失敗は、次の試行のために行を started の
	// まま残すだけにする。
	IsFinalAttempt bool
}

// Execute generates the export's archive and converges its row.
//
// A missing or already terminal export finishes without error: the job carries
// no work that is still outstanding, and returning an error would only retry it
// until the attempts run out.
//
// [Ja] Execute はエクスポートのアーカイブを生成し、その行を収束させる。
//
// エクスポートが存在しない場合、またはすでに終端状態の場合はエラーなしで完了する。
// そのジョブに未処理の作業は残っておらず、エラーを返しても試行を使い切るまで
// リトライされるだけであるため。
func (uc *GenerateExportUsecase) Execute(ctx context.Context, input GenerateExportInput) (err error) {
	export, err := uc.exportRepo.FindByID(ctx, input.ExportID)
	if err != nil {
		return fmt.Errorf("エクスポートの取得に失敗: %w", err)
	}
	if export == nil {
		slog.WarnContext(ctx, "生成対象のエクスポートが見つかりません", "export_id", input.ExportID.String())
		return nil
	}
	if export.Status == model.ExportStatusSucceeded || export.Status == model.ExportStatusFailed {
		slog.InfoContext(ctx, "エクスポートは既に終端状態のため生成をスキップします",
			"export_id", export.ID.String(),
			"status", export.Status.String(),
		)
		return nil
	}

	release, allowed, err := uc.deletionGuard.BeginOperation(ctx, export.ProfileID)
	if err != nil {
		return fmt.Errorf("プロフィールの export 生成 guard の取得に失敗: %w", err)
	}
	if !allowed {
		slog.InfoContext(ctx, "プロフィールが削除中または存在しないためエクスポート生成をスキップします",
			"export_id", export.ID.String(),
			"profile_id", export.ProfileID.String(),
		)
		return nil
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil {
			err = errors.Join(err, fmt.Errorf("プロフィールの export 生成 guard の解放に失敗: %w", releaseErr))
		}
	}()

	started, err := uc.exportRepo.MarkStarted(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		return fmt.Errorf("エクスポートの開始記録に失敗: %w", err)
	}
	if started == nil {
		slog.WarnContext(ctx, "エクスポートの状態が変わったため生成をスキップします", "export_id", export.ID.String())
		return nil
	}

	requester, err := uc.resolveRequester(ctx, started)
	if err != nil {
		return uc.failAttempt(ctx, started, input.IsFinalAttempt, err)
	}

	if err := uc.generate(ctx, started, requester); err != nil {
		return uc.failAttempt(ctx, started, input.IsFinalAttempt, err)
	}
	return nil
}

// failAttempt closes a terminal attempt and otherwise leaves it started for
// River to retry. Every error after MarkStarted passes through here so that an
// accepted attempt is always counted and the final one always converges.
//
// [Ja] failAttempt は最終試行を閉じ、それ以前の試行は River が再試行できるよう
// started のまま残す。受理された試行が必ず計上され、最終試行が必ず収束するよう、
// MarkStarted 後のすべてのエラーをここへ通す。
func (uc *GenerateExportUsecase) failAttempt(ctx context.Context, export *model.Export, isFinalAttempt bool, cause error) error {
	if isFinalAttempt {
		uc.closeAsFailed(ctx, export, cause)
	}
	return cause
}

// exportRequester is the requesting user's presentation context for the
// archive: the locale its own text is written in and the zone its months and
// timestamps are computed in. It follows the requester rather than the
// profile's owner, because the archive is written to be read by whoever asked
// for it, and a profile can be managed by more than one user.
//
// [Ja] exportRequester はアーカイブにとっての申請者の表示コンテキスト。アーカイブ
// 自身の文言を書くロケールと、月と日時を算出するゾーンを持つ。プロフィールの所有者
// ではなく申請者に従うのは、アーカイブが申請した本人に読まれる前提で書かれること、
// および 1 つのプロフィールを複数のユーザーが管理し得ることによる。
type exportRequester struct {
	locale   string
	location *time.Location
}

// resolveRequester resolves the requester's locale and time zone through the
// actor the export was requested with.
//
// [Ja] resolveRequester は、エクスポートを申請した actor を辿って申請者のロケールと
// タイムゾーンを解決する。
func (uc *GenerateExportUsecase) resolveRequester(ctx context.Context, export *model.Export) (exportRequester, error) {
	actor, err := uc.actorRepo.FindByID(ctx, export.ActorID)
	if err != nil {
		return exportRequester{}, fmt.Errorf("アクターの取得に失敗: %w", err)
	}
	if actor == nil {
		return exportRequester{}, fmt.Errorf("エクスポートのアクターが見つからない (export_id: %s)", export.ID.String())
	}

	user, err := uc.userRepo.FindByID(ctx, actor.UserID)
	if err != nil {
		return exportRequester{}, fmt.Errorf("ユーザーの取得に失敗: %w", err)
	}
	if user == nil {
		return exportRequester{}, fmt.Errorf("エクスポートのユーザーが見つからない (export_id: %s)", export.ID.String())
	}

	return exportRequester{
		locale:   user.Locale,
		location: exportLocation(ctx, user),
	}, nil
}

// exportLocation resolves the user's time zone, falling back to UTC when the
// name cannot be resolved. The fallback happens here rather than deeper in the
// flow because the zone is passed on to PostgreSQL, which rejects a location
// that did not come from the zone database (time.Local and time.FixedZone name
// zones it cannot resolve). Failing on that would abort the generation halfway
// instead of producing an archive whose timestamps are merely in UTC.
//
// [Ja] exportLocation はユーザーのタイムゾーンを解決し、名前を解決できない場合は
// UTC へフォールバックする。フォールバックを処理の奥ではなくここで行うのは、
// ゾーンが PostgreSQL へ渡るためである。PostgreSQL はゾーンデータベース由来でない
// location (time.Local や time.FixedZone は解決できない名前を持つ) を拒否する。
// そこで失敗させると、日時が UTC になるだけのアーカイブを作る代わりに、生成を
// 途中で中断することになる。
func exportLocation(ctx context.Context, user *model.User) *time.Location {
	location, err := time.LoadLocation(user.TimeZone)
	if err != nil {
		slog.WarnContext(ctx, "ユーザーのタイムゾーンを解決できないため UTC を使用します",
			"user_id", user.ID.String(),
			"time_zone", user.TimeZone,
			"error", err,
		)
		return time.UTC
	}
	return location
}

// generate streams the archive to the object storage and, once it is stored,
// records the success. The upload runs before the transition, so an attempt
// that dies in between leaves an object at a key the next attempt overwrites
// rather than a success with nothing behind it.
//
// [Ja] generate はアーカイブをオブジェクトストレージへストリーミングし、保存でき
// たら成功を記録する。アップロードを遷移より先に行うため、その間に落ちた試行が
// 残すのは次の試行が上書きするキーのオブジェクトであって、実体のない成功ではない。
func (uc *GenerateExportUsecase) generate(ctx context.Context, export *model.Export, requester exportRequester) error {
	months, err := uc.exportPostRepo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
		ExportID: export.ID,
		Location: requester.location,
	})
	if err != nil {
		return fmt.Errorf("エクスポート対象の月一覧の取得に失敗: %w", err)
	}

	archive := ExportArchive{
		Locale:   requester.locale,
		Location: requester.location,
		Months:   archiveMonths(months),
		// The generation time is stamped on the entries in the requester's zone
		// as well, so a file manager shows the same wall clock as the archive's
		// own pages.
		//
		// [Ja] 生成時刻もエントリへ申請者のゾーンで記録する。ファイルアプリが、
		// アーカイブ自身のページと同じ壁時計を表示するようにするため。
		GeneratedAt: time.Now().In(requester.location),
	}

	objectKey := ExportObjectKey(export.ProfileID, export.ID)
	if err := uc.upload(ctx, objectKey, export.ID, months, archive); err != nil {
		return err
	}

	succeeded, err := uc.exportRepo.MarkSucceeded(ctx, export.ID, objectKey, export.UpdatedAt)
	if err != nil {
		return fmt.Errorf("エクスポートの成功記録に失敗: %w", err)
	}
	if !succeeded {
		// The archive is stored under a key derived from the export, so the
		// attempt that owns the row now writes the same object. Reporting a
		// failure lets the retry re-read the row and finish without work when
		// it has already reached succeeded.
		//
		// [Ja] アーカイブはエクスポートから導かれるキーに保存されているため、今その
		// 行を保持する試行が同じオブジェクトを書く。失敗として報告することで、
		// リトライが行を読み直し、すでに succeeded に達していれば何もせず完了できる。
		return fmt.Errorf("エクスポートの状態が変わったため成功を記録できない (export_id: %s)", export.ID.String())
	}

	slog.InfoContext(ctx, "エクスポートのアーカイブを生成しました",
		"export_id", export.ID.String(),
		"month_count", len(months),
	)

	uc.enqueueCompletionEmail(ctx, export.ID)
	uc.enqueueCleanup(ctx, export.ProfileID)
	return nil
}

// enqueueCompletionEmail asks for the completion notification the success just
// created to be delivered. The recipient is waiting for it, so it is enqueued
// ahead of the cleanup rather than left to the reconciliation that runs every
// few minutes.
//
// A failure is logged rather than returned, for the same reason as the cleanup:
// the retry would read a terminal row and return before reaching this point.
// The notification is a durable row of its own, so reconciliation re-derives
// the delivery from the outbox.
//
// [Ja] enqueueCompletionEmail は、この成功が作成した完了通知の配信を要求する。受信者は
// それを待っているため、数分おきのリコンシリエーションに任せず、cleanup より先に投入
// する。
//
// 失敗は cleanup と同じ理由で、返さずログに残す。リトライは終端状態の行を読み、ここへ
// 到達する前に戻るためである。通知はそれ自体が durable な行なので、リコンシリエーション
// が outbox から配信を再導出する。
func (uc *GenerateExportUsecase) enqueueCompletionEmail(ctx context.Context, exportID model.ExportID) {
	if _, err := uc.dispatcher.EnqueueSendExportCompletedEmail(ctx, exportID.String()); err != nil {
		slog.ErrorContext(ctx, "エクスポート完了メールジョブの投入に失敗しました",
			"export_id", exportID.String(),
			"error", err,
		)
	}
}

// enqueueCleanup asks for the exports this success replaced to be deleted. The
// new archive is already the only one offered for download, so the deletion is
// about the storage the previous ones still occupy.
//
// A failure is logged rather than returned. The archive is stored and the row
// is succeeded, so reporting one would retry an attempt with nothing left to
// do: the retry reads a terminal row and returns without reaching this point,
// which would lose the cleanup rather than repeat it. Reconciliation re-derives
// it instead, from the profiles that hold a succeeded export older than their
// latest one.
//
// [Ja] enqueueCleanup は、この成功が置き換えたエクスポートの削除を要求する。新しい
// アーカイブはすでに唯一のダウンロード対象であるため、この削除は以前のものが占有し
// 続けるストレージのためのものである。
//
// 失敗は返さずログに残す。アーカイブは保存済みで行は succeeded のため、失敗として
// 報告しても、やることが残っていない試行を再試行するだけになる。リトライは終端状態の
// 行を読んでここへ到達せずに戻るので、掃除は繰り返されるのではなく失われる。代わりに
// リコンシリエーションが、最新より古い succeeded を持つプロフィールから再導出する。
func (uc *GenerateExportUsecase) enqueueCleanup(ctx context.Context, profileID model.ProfileID) {
	if _, err := uc.dispatcher.EnqueueCleanupOldExports(ctx, profileID.String()); err != nil {
		slog.ErrorContext(ctx, "旧エクスポート削除ジョブの投入に失敗しました",
			"profile_id", profileID.String(),
			"error", err,
		)
	}
}

// upload writes the archive into a pipe that the object storage reads from, so
// the zip is uploaded while it is still being built and neither a month nor the
// whole archive is ever held in memory.
//
// [Ja] upload はアーカイブを pipe へ書き出し、オブジェクトストレージがそこから
// 読み取る。これにより zip は構築中と並行してアップロードされ、月全体もアーカイブ
// 全体もメモリに保持されない。
func (uc *GenerateExportUsecase) upload(
	ctx context.Context,
	objectKey string,
	exportID model.ExportID,
	months []repository.PostMonth,
	archive ExportArchive,
) error {
	pr, pw := io.Pipe()
	// Release the archive goroutine even when this function leaves through a
	// panic raised by the upload: with no reader left, the goroutine would block
	// in Write for as long as the process lives. Closing twice is a no-op, so
	// the ordered close below still decides what the writer is handed.
	//
	// [Ja] アップロードの panic で本関数を抜けた場合でも、アーカイブの goroutine を
	// 解放する。読み手がいなくなると、その goroutine はプロセスが生きている限り
	// Write でブロックし続けるため。2 回閉じても何も起きないので、writer が受け取る
	// エラーは下の順序どおりの close が決める。
	defer func() { _ = pr.CloseWithError(errExportUploadStopped) }()

	writeDone := make(chan error, 1)
	go func() {
		var err error
		// Recover here instead of letting the panic escape. The job queue
		// recovers a panic raised in the worker's own goroutine, but not one
		// raised in a goroutine that worker started, so an escaping panic would
		// end the whole process rather than this attempt.
		//
		// [Ja] panic を外へ逃がさずここで recover する。ジョブキューが recover する
		// のは worker 自身の goroutine で起きた panic であり、worker が起動した
		// goroutine のものは recover しない。逃がすとこの試行ではなくプロセス全体が
		// 終わることになる。
		defer func() {
			if recovered := recover(); recovered != nil {
				// The stack goes to the log rather than into the error, because
				// the error becomes the attempt's recorded failure and the text
				// of the issue it raises, both of which stay readable only while
				// they hold the panic value alone.
				//
				// [Ja] スタックはエラーではなくログへ出す。エラーは試行の失敗記録と、
				// そこから起票される issue の文言になるため、panic 値だけを持つ状態に
				// 保たないと読めなくなる。
				slog.ErrorContext(ctx, "エクスポートアーカイブの書き出しで panic が発生しました",
					"export_id", exportID.String(),
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				err = fmt.Errorf("エクスポートアーカイブの書き出しで panic が発生: %v", recovered)
			}
			// Closing with the build error hands it to the reader, so the upload
			// stops instead of storing a truncated archive.
			//
			// [Ja] 構築時のエラーで閉じることでそれを読み手へ渡し、切り詰められた
			// アーカイブを保存せずにアップロードを止める。
			_ = pw.CloseWithError(err)
			writeDone <- err
		}()
		err = uc.writeArchive(ctx, pw, exportID, months, archive)
	}()

	uploadErr := uc.objectStorage.Upload(ctx, objectKey, pr)
	// Close the read side before waiting: an upload that stopped early leaves
	// the archive goroutine blocked in Write, and it only returns once the
	// reader is gone.
	//
	// [Ja] 待つ前に読み取り側を閉じる。途中で終了したアップロードはアーカイブの
	// goroutine を Write でブロックしたままにし、読み手がいなくなって初めて戻る
	// ため。
	_ = pr.CloseWithError(errExportUploadStopped)
	writeErr := <-writeDone
	if errors.Is(writeErr, errExportUploadStopped) {
		// The archive stopped because the upload was already over, so this is
		// the close above coming back rather than a failure of its own.
		//
		// [Ja] アーカイブが止まったのはアップロードが既に終わっていたからで、これは
		// 上の close が返ってきたものであって、それ自体の失敗ではない。
		writeErr = nil
	}

	if writeErr != nil || uploadErr != nil {
		// The build error comes first: when it is what stopped the upload, the
		// upload error is only the pipe reporting it back.
		//
		// [Ja] 構築時のエラーを先に置く。それがアップロードを止めた場合、
		// アップロード側のエラーは pipe がそれを返しているだけであるため。
		return fmt.Errorf(
			"エクスポートアーカイブのアップロードに失敗 (key: %s): %w",
			objectKey,
			errors.Join(writeErr, uploadErr),
		)
	}
	return nil
}

// writeArchive writes index.html and then one entry per month, reading each
// month's posts a page at a time so the memory held stays proportional to a
// page rather than to the export.
//
// [Ja] writeArchive は index.html を書き出し、続いて月ごとに 1 つのエントリを
// 書き出す。各月の投稿はページ単位で読むため、保持するメモリはエクスポート全体
// ではなく 1 ページに比例する。
func (uc *GenerateExportUsecase) writeArchive(
	ctx context.Context,
	w io.Writer,
	exportID model.ExportID,
	months []repository.PostMonth,
	archive ExportArchive,
) error {
	writer := uc.archiveBuilder.NewArchive(w, archive)
	// Close is idempotent, so the deferred call only matters on the paths that
	// return early: it releases the writer without leaving an entry open. Its
	// error is logged rather than returned, because on those paths the error
	// that stopped the archive is the one the caller has to see. The stop error
	// is left out of that log: it says the upload had already closed the pipe,
	// which is how every failed upload ends rather than a failure of its own.
	//
	// [Ja] Close は冪等のため、defer した呼び出しが意味を持つのは途中で return
	// する経路だけ。エントリを開いたままにせず writer を解放する。そのエラーは返さず
	// ログに残す。これらの経路で呼び出し側が見るべきなのは、アーカイブを止めたほうの
	// エラーであるため。ただし中断用のエラーはログから除く。アップロードが既に pipe を
	// 閉じたことを示すもので、失敗したアップロードは必ずこうなる。それ自体の失敗では
	// ないため。
	defer func() {
		if err := writer.Close(); err != nil && !errors.Is(err, errExportUploadStopped) {
			slog.WarnContext(ctx, "中断したエクスポートアーカイブのクローズに失敗しました",
				"export_id", exportID.String(),
				"error", err,
			)
		}
	}()

	if err := writer.WriteIndex(ctx); err != nil {
		return fmt.Errorf("エクスポートの目次の書き出しに失敗: %w", err)
	}

	for i, month := range months {
		if err := uc.writeMonth(ctx, writer, exportID, archive.Months[i], month); err != nil {
			return err
		}
	}

	// Close explicitly as well, because it is what verifies that the archive is
	// complete; leaving it to the deferred call would discard that verdict.
	//
	// [Ja] 明示的にも Close する。アーカイブが完成しているかを検証するのが Close で
	// あり、defer した呼び出しに任せるとその判定を捨てることになるため。
	if err := writer.Close(); err != nil {
		return fmt.Errorf("エクスポートアーカイブのクローズに失敗: %w", err)
	}
	return nil
}

// writeMonth writes one month's entry, walking the export's post snapshot with
// the cursor the repository returns.
//
// [Ja] writeMonth は 1 か月分のエントリを書き出す。エクスポートの投稿 snapshot は
// repository が返す cursor で辿る。
func (uc *GenerateExportUsecase) writeMonth(
	ctx context.Context,
	writer ExportArchiveWriter,
	exportID model.ExportID,
	archiveMonth ExportArchiveMonth,
	month repository.PostMonth,
) error {
	monthWriter, err := writer.OpenMonth(ctx, archiveMonth)
	if err != nil {
		return fmt.Errorf("エクスポートの月のエントリの作成に失敗: %w", err)
	}
	// As in writeArchive, the deferred Close only covers the paths that return
	// early, its error is logged so that the one that stopped the month stays
	// the error the caller sees, and the stop error is left out of that log.
	//
	// [Ja] writeArchive と同様、defer した Close が担うのは途中で return する経路
	// だけで、そのエラーはログに残す。呼び出し側が見るエラーを、月の書き出しを止めた
	// ほうに保つため。中断用のエラーをログから除くのも同様。
	defer func() {
		if err := monthWriter.Close(); err != nil && !errors.Is(err, errExportUploadStopped) {
			slog.WarnContext(ctx, "中断したエクスポートの月のエントリのクローズに失敗しました",
				"export_id", exportID.String(),
				"error", err,
			)
		}
	}()

	var cursor *repository.PostCursor
	for {
		posts, next, err := uc.exportPostRepo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
			ExportID: exportID,
			Month:    month,
			Cursor:   cursor,
			PageSize: ExportPostPageSize,
		})
		if err != nil {
			return fmt.Errorf("エクスポート対象の投稿の取得に失敗: %w", err)
		}

		for _, post := range posts {
			if err := monthWriter.WritePost(ctx, ExportArchivePost{
				ID:          post.ID.String(),
				Content:     post.Content,
				PublishedAt: post.PublishedAt,
			}); err != nil {
				return fmt.Errorf("エクスポートの投稿の書き出しに失敗: %w", err)
			}
		}

		if next == nil {
			break
		}
		cursor = next
	}

	// Closing the month verifies that as many posts were written as the index
	// declared, so its error is a build failure rather than a detail of the
	// deferred cleanup.
	//
	// [Ja] 月を閉じる処理は、目次が宣言した件数だけ投稿を書き出したかを検証する。
	// そのエラーは defer した後始末の細部ではなく、構築の失敗として扱う。
	if err := monthWriter.Close(); err != nil {
		return fmt.Errorf("エクスポートの月のエントリのクローズに失敗: %w", err)
	}
	return nil
}

// closeAsFailed ends an export whose last attempt failed, so the screen shows
// a failure the user can retry from instead of a generation that never
// finishes. It runs on a context detached from the attempt's, which the failure
// may well have canceled, and bounded by its own timeout.
//
// The row is closed before the object is removed: the transition proves this
// attempt still owns the export, so the removal cannot take an archive a
// concurrent success published under the same key. An object left behind by a
// failure between the two is collected by the orphan sweep.
//
// [Ja] closeAsFailed は最終試行が失敗したエクスポートを終わらせ、画面に、終わらない
// 生成ではなくユーザーが再実行できる失敗を表示させる。処理は、その失敗によって
// キャンセルされている可能性が高い試行の context から切り離し、専用の timeout で
// 区切った context で実行する。
//
// オブジェクトの削除より先に行を閉じる。遷移が成功することでこの試行がまだ
// エクスポートを保持していると分かるため、同じキーへ並行する成功が公開した
// アーカイブを削除で奪うことがない。2 つの処理の間の失敗で残ったオブジェクトは
// 孤児回収が回収する。
func (uc *GenerateExportUsecase) closeAsFailed(ctx context.Context, export *model.Export, cause error) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), exportTerminalCleanupTimeout)
	defer cancel()

	failed, err := uc.exportRepo.MarkFailed(cleanupCtx, export.ID, export.UpdatedAt)
	if err != nil {
		slog.ErrorContext(cleanupCtx, "エクスポートの失敗記録に失敗しました",
			"export_id", export.ID.String(),
			"error", err,
		)
		return
	}
	if !failed {
		slog.WarnContext(cleanupCtx, "エクスポートの状態が変わったため失敗を記録しませんでした", "export_id", export.ID.String())
		return
	}

	slog.WarnContext(cleanupCtx, "エクスポートの生成を失敗として終了しました",
		"export_id", export.ID.String(),
		"error", cause,
	)

	objectKey := ExportObjectKey(export.ProfileID, export.ID)
	if err := uc.objectStorage.Delete(cleanupCtx, objectKey); err != nil {
		// The object is not referenced by any row now, so leaving it costs
		// storage until the orphan sweep removes it. That is a better outcome
		// than reporting the export as still running.
		//
		// [Ja] このオブジェクトはどの行からも参照されないため、残しても孤児回収が
		// 消すまでのストレージを使うだけで済む。エクスポートが実行中のままだと
		// 報告するよりは良い結果になる。
		slog.ErrorContext(cleanupCtx, "失敗したエクスポートのオブジェクト削除に失敗しました",
			"export_id", export.ID.String(),
			"error", err,
		)
	}
}

// archiveMonths converts the repository's months into the months the archive
// declares. The two slices stay in the same order, so the entry the builder
// opens and the range the posts are read from always describe one month.
//
// [Ja] archiveMonths は repository の月を、アーカイブが宣言する月へ変換する。
// 2 つのスライスは同じ順序を保つため、builder が開くエントリと投稿を読み取る範囲は
// 常に同じ月を指す。
func archiveMonths(months []repository.PostMonth) []ExportArchiveMonth {
	archived := make([]ExportArchiveMonth, len(months))
	for i, month := range months {
		archived[i] = ExportArchiveMonth{
			LocalMonthStart: month.LocalMonthStart,
			PostCount:       month.PostCount,
		}
	}
	return archived
}
