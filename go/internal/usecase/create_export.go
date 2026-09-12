package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// CreateExportUsecase starts an export for the signed-in profile. It persists
// the request as a queued row and then asks the job queue to generate it.
//
// The row is the durable record of the request, not the job. A queued row and
// its River job cannot be written atomically (the job queue runs on its own
// connection), so the insert commits first and the job is inserted after. An
// insert that fails leaves the queued row in place for reconciliation to pick
// up, which is why a failed enqueue does not fail the request.
//
// [Ja] CreateExportUsecase はログイン中プロフィールのエクスポートを開始する。
// 申請を queued の行として永続化し、その後ジョブキューへ生成を依頼する。
//
// 申請の永続的な記録はジョブではなく行のほうである。queued の行と River のジョブは
// 原子的に書けない (ジョブキューは自身のコネクションで動く) ため、先に INSERT を
// commit し、ジョブはその後に投入する。投入が失敗しても queued の行は残り、
// リコンシリエーションが引き受ける。投入の失敗でリクエストを失敗させないのは
// このためである。
type CreateExportUsecase struct {
	db              *sql.DB
	userProfileRepo *repository.UserProfileRepository
	exportRepo      *repository.ExportRepository
	dispatcher      *dispatcher.Dispatcher
	storageReady    bool
}

// NewCreateExportUsecase creates a CreateExportUsecase. storageReady is the
// same readiness the export Workers and the export page are gated on, resolved
// once at composition time.
//
// [Ja] NewCreateExportUsecase は CreateExportUsecase を生成する。storageReady は
// エクスポート系 Worker とエクスポート画面と同じ readiness で、合成時に一度だけ
// 解決した値である。
func NewCreateExportUsecase(
	db *sql.DB,
	userProfileRepo *repository.UserProfileRepository,
	exportRepo *repository.ExportRepository,
	d *dispatcher.Dispatcher,
	storageReady bool,
) *CreateExportUsecase {
	return &CreateExportUsecase{
		db:              db,
		userProfileRepo: userProfileRepo,
		exportRepo:      exportRepo,
		dispatcher:      d,
		storageReady:    storageReady,
	}
}

// CreateExportInput holds the input parameters for starting an export. UserID
// and ProfileID are the pair authorization is decided on, exactly as on the
// export page. ActorID is the requester recorded on the row; it is not part of
// the authorization decision.
//
// [Ja] CreateExportInput はエクスポート開始の入力パラメータ。UserID と ProfileID は
// エクスポート画面と同じく認可を判断する組である。ActorID は行に記録する申請者で
// あり、認可の判断には加わらない。
type CreateExportInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
	ActorID   model.ActorID
}

// CreateExportOutput is the result of starting an export. Export is the queued
// row that was created.
//
// [Ja] CreateExportOutput はエクスポート開始の結果。Export は作成された queued の
// 行である。
type CreateExportOutput struct {
	Export *model.Export
}

// Execute authorizes the request, persists the queued export, and asks the job
// queue to generate it.
//
// Authorization runs before the availability check for the same reason as on
// the export page: a profile the user does not own is refused identically
// whether or not this deployment can run exports. A deployment that cannot is
// then refused before anything is written, so no row is created that no Worker
// exists to generate.
//
// [Ja] Execute はリクエストを認可し、queued のエクスポートを永続化し、ジョブキューへ
// 生成を依頼する。
//
// 認可を利用可否の判定より先に行う理由はエクスポート画面と同じで、ユーザーが所有して
// いないプロフィールを、そのデプロイがエクスポートを実行できるかどうかに関わらず
// 同じ形で拒否するためである。実行できないデプロイは、何かを書き込む前に拒否される
// ため、生成する Worker が存在しない行が作られることはない。
func (uc *CreateExportUsecase) Execute(ctx context.Context, input CreateExportInput) (*CreateExportOutput, error) {
	if err := uc.authorize(ctx, input); err != nil {
		return nil, err
	}

	if !uc.storageReady {
		return nil, &model.AppError{
			Code:     model.AppErrCodeServiceUnavailable,
			UserMsg:  i18n.T(ctx, "error_export_unavailable"),
			Internal: errors.New("エクスポート用オブジェクトストレージが未設定"),
			Metadata: map[string]string{"profile_id": input.ProfileID.String()},
		}
	}

	export, err := uc.createExport(ctx, input)
	if err != nil {
		return nil, err
	}

	uc.enqueueGeneration(ctx, export.ID)

	return &CreateExportOutput{Export: export}, nil
}

// authorize checks that the signed-in user currently owns the target profile,
// on the same terms as GetExportShowUsecase: the right to export a profile's
// posts comes from the ownership the user holds right now, not from the actor
// row that records who asked. A profile the user does not own is refused as not
// found, so the response cannot be used to tell an existing profile from a
// missing one.
//
// [Ja] authorize はログイン中ユーザーが対象プロフィールを現在所有していることを
// 確認する。条件は GetExportShowUsecase と同じで、プロフィールのポストを
// エクスポートする権利は、誰が申請したかを記録する actor 行ではなく、そのユーザーが
// 今持っている所有関係から生じる。所有していないプロフィールは not found として
// 拒否し、応答から既存のプロフィールと存在しないプロフィールを区別できないようにする。
func (uc *CreateExportUsecase) authorize(ctx context.Context, input CreateExportInput) error {
	userProfile, err := uc.userProfileRepo.FindByProfileID(ctx, input.ProfileID)
	if err != nil {
		return fmt.Errorf("プロフィールの所有関係の取得に失敗: %w", err)
	}

	if userProfile == nil || userProfile.UserID != input.UserID {
		return &model.AppError{
			Code:    model.AppErrCodeResourceNotFound,
			UserMsg: i18n.T(ctx, "error_not_found_message"),
		}
	}

	return nil
}

// createExport removes the profile's failed exports and inserts the new queued
// one in a single transaction.
//
// Both changes belong to the same commit: the failed row is only obsolete
// because a new request replaces it, so a create that loses the race for the
// profile's one active slot must leave it standing. Rolling back together also
// keeps the retention bound intact, since the profile never holds two exports
// that are neither succeeded nor in progress.
//
// [Ja] createExport はプロフィールの failed なエクスポートの削除と、新しい queued の
// エクスポートの挿入を 1 つの transaction で行う。
//
// 2 つの変更は同じ commit に属する。failed の行が不要になるのは新しい申請がそれを
// 置き換えるからであり、プロフィールの 1 つしかない実行枠を取り損ねた create は、
// その行を残さなければならない。まとめて rollback することで保持の上限も保たれる。
// succeeded でも進行中でもないエクスポートをプロフィールが 2 件持つことがなくなる
// ためである。
func (uc *CreateExportUsecase) createExport(ctx context.Context, input CreateExportInput) (*model.Export, error) {
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	exportRepo := uc.exportRepo.WithTx(tx)

	if _, err := exportRepo.DeleteFailedByProfileID(ctx, input.ProfileID); err != nil {
		return nil, fmt.Errorf("失敗したエクスポートの削除に失敗: %w", err)
	}

	export, err := exportRepo.Create(ctx, repository.CreateExportInput{
		ProfileID: input.ProfileID,
		ActorID:   input.ActorID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrActiveExportExists) {
			return nil, &model.AppError{
				Code:     model.AppErrCodeConflict,
				UserMsg:  i18n.T(ctx, "flash_export_already_in_progress"),
				Internal: err,
				Metadata: map[string]string{"profile_id": input.ProfileID.String()},
			}
		}
		return nil, fmt.Errorf("エクスポートの作成に失敗: %w", err)
	}
	// No row means the profile is past the boundary profile deletion
	// establishes. Nothing failed, so this is a refusal rather than an error:
	// the posts an export would archive are on their way out.
	//
	// [Ja] 行が無いのは、プロフィールが削除処理の確立する境界を越えた場合である。
	// 失敗ではないため、エラーではなく拒否として扱う。エクスポートがアーカイブする
	// はずのポストは、これから消えていくものであるため。
	if export == nil {
		return nil, &model.AppError{
			Code:     model.AppErrCodeConflict,
			UserMsg:  i18n.T(ctx, "flash_export_profile_deleting"),
			Internal: errors.New("プロフィールの削除が開始済みのためエクスポートを作成しない"),
			Metadata: map[string]string{"profile_id": input.ProfileID.String()},
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return export, nil
}

// enqueueGeneration asks the job queue to generate the committed export. A
// failure is logged and swallowed: the queued row is the durable work intent,
// and reconciliation inserts a job for a queued export that never got one, so
// reporting the request as failed here would deny an export that is in fact
// going to run.
//
// [Ja] enqueueGeneration は commit 済みのエクスポートの生成をジョブキューへ依頼する。
// 失敗はログに記録して握りつぶす。queued の行が durable work intent であり、ジョブを
// 得られなかった queued のエクスポートにはリコンシリエーションがジョブを投入するため、
// ここでリクエストを失敗として報告すると、実際には実行されるエクスポートを拒否する
// ことになる。
func (uc *CreateExportUsecase) enqueueGeneration(ctx context.Context, exportID model.ExportID) {
	if _, err := uc.dispatcher.EnqueueGenerateExport(ctx, exportID.String()); err != nil {
		slog.WarnContext(ctx, "エクスポート生成ジョブの投入に失敗", "error", err, "export_id", exportID.String())
	}
}
