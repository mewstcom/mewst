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

// exportTerminalCleanupTimeoutは最終試行が失敗した後の後処理の上限時間。
// 後処理はジョブのcontext (すでにキャンセル済みまたはtimeout済みのことがある)
// から切り離したcontextで実行するため、このtimeoutが、試行が終わった後の
// workerを後処理でハングさせないための歯止めになる。
const exportTerminalCleanupTimeout = 30 * time.Second

// errExportUploadStoppedはuploadが読み取りを止めたときにアーカイブ側の
// pipeを閉じる。いなくなった読み手を永久に待ち続けるwriterを解放するためだけの
// ものなので、uploadは報告するエラーから取り除く。実際にアップロードを止めた失敗の
// 前に、これを並べないようにするため。
var errExportUploadStopped = errors.New("アップロードが終了したためアーカイブの書き出しを中断した")

// GenerateExportUsecaseは1件のエクスポートのzipを構築し、オブジェクト
// ストレージへストリーミングする。1回の試行全体を担い、エクスポートをqueued
// (リトライが再入する場合はstartedの行) からアップロードを経てsucceededまで
// 進める。最終試行では、startedのまま放置せずfailedとしてエクスポートを閉じる。
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

// NewGenerateExportUsecaseはGenerateExportUsecaseを生成する。
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

// GenerateExportInputは1件のエクスポートを生成する入力パラメータ。
type GenerateExportInput struct {
	// ExportIDは生成対象のエクスポート。
	ExportID model.ExportID

	// IsFinalAttemptはこの試行の失敗でエクスポートが終わるかどうかを表す。
	// 再試行の予算はジョブキューが持つため、本UseCaseが導出し直すのではなく
	// 呼び出し側が判定を渡す。最終試行ではエクスポートをfailedとして閉じ、その
	// オブジェクトを削除する。それ以前の失敗は、次の試行のために行をstartedの
	// まま残すだけにする。
	IsFinalAttempt bool
}

// Executeはエクスポートのアーカイブを生成し、その行を収束させる。
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
		return fmt.Errorf("プロフィールのexport生成guardの取得に失敗: %w", err)
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
			err = errors.Join(err, fmt.Errorf("プロフィールのexport生成guardの解放に失敗: %w", releaseErr))
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

// failAttemptは最終試行を閉じ、それ以前の試行はRiverが再試行できるよう
// startedのまま残す。受理された試行が必ず計上され、最終試行が必ず収束するよう、
// MarkStarted後のすべてのエラーをここへ通す。
func (uc *GenerateExportUsecase) failAttempt(ctx context.Context, export *model.Export, isFinalAttempt bool, cause error) error {
	if isFinalAttempt {
		uc.closeAsFailed(ctx, export, cause)
	}
	return cause
}

// exportRequesterはアーカイブにとっての申請者の表示コンテキスト。アーカイブ
// 自身の文言を書くロケールと、月と日時を算出するゾーンを持つ。プロフィールの所有者
// ではなく申請者に従うのは、アーカイブが申請した本人に読まれる前提で書かれること、
// および1つのプロフィールを複数のユーザーが管理し得ることによる。
type exportRequester struct {
	locale   string
	location *time.Location
}

// resolveRequesterは、エクスポートを申請したactorを辿って申請者のロケールと
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

// exportLocationはユーザーのタイムゾーンを解決し、名前を解決できない場合は
// UTCへフォールバックする。フォールバックを処理の奥ではなくここで行うのは、
// ゾーンがPostgreSQLへ渡るためである。PostgreSQLはゾーンデータベース由来でない
// location (time.Localやtime.FixedZoneは解決できない名前を持つ) を拒否する。
// そこで失敗させると、日時がUTCになるだけのアーカイブを作る代わりに、生成を
// 途中で中断することになる。
func exportLocation(ctx context.Context, user *model.User) *time.Location {
	location, err := time.LoadLocation(user.TimeZone)
	if err != nil {
		slog.WarnContext(ctx, "ユーザーのタイムゾーンを解決できないためUTCを使用します",
			"user_id", user.ID.String(),
			"time_zone", user.TimeZone,
			"error", err,
		)
		return time.UTC
	}
	return location
}

// generateはアーカイブをオブジェクトストレージへストリーミングし、保存でき
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
		// 生成時刻もエントリへ申請者のゾーンで記録する。ファイルアプリが、
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
		// アーカイブはエクスポートから導かれるキーに保存されているため、今その
		// 行を保持する試行が同じオブジェクトを書く。失敗として報告することで、
		// リトライが行を読み直し、すでにsucceededに達していれば何もせず完了できる。
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

// enqueueCompletionEmailは、この成功が作成した完了通知の配信を要求する。受信者は
// それを待っているため、数分おきのリコンシリエーションに任せず、cleanupより先に投入
// する。
//
// 失敗はcleanupと同じ理由で、返さずログに残す。リトライは終端状態の行を読み、ここへ
// 到達する前に戻るためである。通知はそれ自体がdurableな行なので、リコンシリエーション
// がoutboxから配信を再導出する。
func (uc *GenerateExportUsecase) enqueueCompletionEmail(ctx context.Context, exportID model.ExportID) {
	if _, err := uc.dispatcher.EnqueueSendExportCompletedEmail(ctx, exportID.String()); err != nil {
		slog.ErrorContext(ctx, "エクスポート完了メールジョブの投入に失敗しました",
			"export_id", exportID.String(),
			"error", err,
		)
	}
}

// enqueueCleanupは、この成功が置き換えたエクスポートの削除を要求する。新しい
// アーカイブはすでに唯一のダウンロード対象であるため、この削除は以前のものが占有し
// 続けるストレージのためのものである。
//
// 失敗は返さずログに残す。アーカイブは保存済みで行はsucceededのため、失敗として
// 報告しても、やることが残っていない試行を再試行するだけになる。リトライは終端状態の
// 行を読んでここへ到達せずに戻るので、掃除は繰り返されるのではなく失われる。代わりに
// リコンシリエーションが、最新より古いsucceededを持つプロフィールから再導出する。
func (uc *GenerateExportUsecase) enqueueCleanup(ctx context.Context, profileID model.ProfileID) {
	if _, err := uc.dispatcher.EnqueueCleanupOldExports(ctx, profileID.String()); err != nil {
		slog.ErrorContext(ctx, "旧エクスポート削除ジョブの投入に失敗しました",
			"profile_id", profileID.String(),
			"error", err,
		)
	}
}

// uploadはアーカイブをpipeへ書き出し、オブジェクトストレージがそこから
// 読み取る。これによりzipは構築中と並行してアップロードされ、月全体もアーカイブ
// 全体もメモリに保持されない。
func (uc *GenerateExportUsecase) upload(
	ctx context.Context,
	objectKey string,
	exportID model.ExportID,
	months []repository.PostMonth,
	archive ExportArchive,
) error {
	pr, pw := io.Pipe()
	// アップロードのpanicで本関数を抜けた場合でも、アーカイブのgoroutineを
	// 解放する。読み手がいなくなると、そのgoroutineはプロセスが生きている限り
	// Writeでブロックし続けるため。2回閉じても何も起きないので、writerが受け取る
	// エラーは下の順序どおりのcloseが決める。
	defer func() { _ = pr.CloseWithError(errExportUploadStopped) }()

	writeDone := make(chan error, 1)
	go func() {
		var err error
		// panicを外へ逃がさずここでrecoverする。ジョブキューがrecoverする
		// のはworker自身のgoroutineで起きたpanicであり、workerが起動した
		// goroutineのものはrecoverしない。逃がすとこの試行ではなくプロセス全体が
		// 終わることになる。
		defer func() {
			if recovered := recover(); recovered != nil {
				// スタックはエラーではなくログへ出す。エラーは試行の失敗記録と、
				// そこから起票されるissueの文言になるため、panic値だけを持つ状態に
				// 保たないと読めなくなる。
				slog.ErrorContext(ctx, "エクスポートアーカイブの書き出しでpanicが発生しました",
					"export_id", exportID.String(),
					"panic", recovered,
					"stack", string(debug.Stack()),
				)
				err = fmt.Errorf("エクスポートアーカイブの書き出しでpanicが発生: %v", recovered)
			}
			// 構築時のエラーで閉じることでそれを読み手へ渡し、切り詰められた
			// アーカイブを保存せずにアップロードを止める。
			_ = pw.CloseWithError(err)
			writeDone <- err
		}()
		err = uc.writeArchive(ctx, pw, exportID, months, archive)
	}()

	uploadErr := uc.objectStorage.Upload(ctx, objectKey, pr)
	// 待つ前に読み取り側を閉じる。途中で終了したアップロードはアーカイブの
	// goroutineをWriteでブロックしたままにし、読み手がいなくなって初めて戻る
	// ため。
	_ = pr.CloseWithError(errExportUploadStopped)
	writeErr := <-writeDone
	if errors.Is(writeErr, errExportUploadStopped) {
		// アーカイブが止まったのはアップロードが既に終わっていたからで、これは
		// 上のcloseが返ってきたものであって、それ自体の失敗ではない。
		writeErr = nil
	}

	if writeErr != nil || uploadErr != nil {
		// 構築時のエラーを先に置く。それがアップロードを止めた場合、
		// アップロード側のエラーはpipeがそれを返しているだけであるため。
		return fmt.Errorf(
			"エクスポートアーカイブのアップロードに失敗 (key: %s): %w",
			objectKey,
			errors.Join(writeErr, uploadErr),
		)
	}
	return nil
}

// writeArchiveはindex.htmlを書き出し、続いて月ごとに1つのエントリを
// 書き出す。各月の投稿はページ単位で読むため、保持するメモリはエクスポート全体
// ではなく1ページに比例する。
func (uc *GenerateExportUsecase) writeArchive(
	ctx context.Context,
	w io.Writer,
	exportID model.ExportID,
	months []repository.PostMonth,
	archive ExportArchive,
) error {
	writer := uc.archiveBuilder.NewArchive(w, archive)
	// Closeは冪等のため、deferした呼び出しが意味を持つのは途中でreturn
	// する経路だけ。エントリを開いたままにせずwriterを解放する。そのエラーは返さず
	// ログに残す。これらの経路で呼び出し側が見るべきなのは、アーカイブを止めたほうの
	// エラーであるため。ただし中断用のエラーはログから除く。アップロードが既にpipeを
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

	// 明示的にもCloseする。アーカイブが完成しているかを検証するのがCloseで
	// あり、deferした呼び出しに任せるとその判定を捨てることになるため。
	if err := writer.Close(); err != nil {
		return fmt.Errorf("エクスポートアーカイブのクローズに失敗: %w", err)
	}
	return nil
}

// writeMonthは1か月分のエントリを書き出す。エクスポートの投稿snapshotは
// repositoryが返すcursorで辿る。
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
	// writeArchiveと同様、deferしたCloseが担うのは途中でreturnする経路
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

	// 月を閉じる処理は、目次が宣言した件数だけ投稿を書き出したかを検証する。
	// そのエラーはdeferした後始末の細部ではなく、構築の失敗として扱う。
	if err := monthWriter.Close(); err != nil {
		return fmt.Errorf("エクスポートの月のエントリのクローズに失敗: %w", err)
	}
	return nil
}

// closeAsFailedは最終試行が失敗したエクスポートを終わらせ、画面に、終わらない
// 生成ではなくユーザーが再実行できる失敗を表示させる。処理は、その失敗によって
// キャンセルされている可能性が高い試行のcontextから切り離し、専用のtimeoutで
// 区切ったcontextで実行する。
//
// オブジェクトの削除より先に行を閉じる。遷移が成功することでこの試行がまだ
// エクスポートを保持していると分かるため、同じキーへ並行する成功が公開した
// アーカイブを削除で奪うことがない。2つの処理の間の失敗で残ったオブジェクトは
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
		// このオブジェクトはどの行からも参照されないため、残しても孤児回収が
		// 消すまでのストレージを使うだけで済む。エクスポートが実行中のままだと
		// 報告するよりは良い結果になる。
		slog.ErrorContext(cleanupCtx, "失敗したエクスポートのオブジェクト削除に失敗しました",
			"export_id", export.ID.String(),
			"error", err,
		)
	}
}

// archiveMonthsはrepositoryの月を、アーカイブが宣言する月へ変換する。
// 2つのスライスは同じ順序を保つため、builderが開くエントリと投稿を読み取る範囲は
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
