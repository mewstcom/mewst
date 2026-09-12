package usecase

import (
	"context"
	"fmt"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// GetExportShowUsecase reads what the export page (GET /settings/export) shows
// for the signed-in profile: the latest export, which drives the state message,
// and the latest succeeded export, which decides whether a zip can be
// downloaded. The two are read separately because a newer in-progress or failed
// export must not hide the zip an earlier export already produced.
//
// [Ja] GetExportShowUsecase はエクスポート画面 (GET /settings/export) がログイン中
// プロフィールについて表示する内容を取得する。状態メッセージを駆動する最新の
// エクスポートと、zip をダウンロードできるかを決める最新の成功したエクスポートを
// 読む。2 つを分けて読むのは、より新しい進行中・失敗のエクスポートが、以前の
// エクスポートが既に作った zip を隠さないようにするため。
type GetExportShowUsecase struct {
	userProfileRepo *repository.UserProfileRepository
	exportRepo      *repository.ExportRepository
	storageReady    bool
}

// NewGetExportShowUsecase creates a GetExportShowUsecase. storageReady tells it
// whether this deployment has the export object storage configured; the value
// is resolved once at composition time from the same readiness the export
// Workers are gated on.
//
// [Ja] NewGetExportShowUsecase は GetExportShowUsecase を生成する。storageReady は
// そのデプロイでエクスポート用オブジェクトストレージが設定されているかを表し、
// エクスポート系 Worker の登録と同じ readiness から合成時に一度だけ解決する。
func NewGetExportShowUsecase(
	userProfileRepo *repository.UserProfileRepository,
	exportRepo *repository.ExportRepository,
	storageReady bool,
) *GetExportShowUsecase {
	return &GetExportShowUsecase{
		userProfileRepo: userProfileRepo,
		exportRepo:      exportRepo,
		storageReady:    storageReady,
	}
}

// GetExportShowInput holds the input parameters for reading the export page.
// UserID and ProfileID both come from the session: the export target is the
// profile, and the user is who must currently own it.
//
// [Ja] GetExportShowInput はエクスポート画面の取得の入力パラメータ。UserID と
// ProfileID はどちらもセッション由来で、エクスポート対象はプロフィール、
// ユーザーはそのプロフィールを現在所有していなければならない側を表す。
type GetExportShowInput struct {
	UserID    model.UserID
	ProfileID model.ProfileID
}

// GetExportShowOutput is the result of reading the export page. Both exports
// are nil when the profile has none, and both are nil when Available is false
// (an unavailable deployment shows no export state at all).
//
// [Ja] GetExportShowOutput はエクスポート画面の取得結果。プロフィールが
// エクスポートを持たない場合、および Available が false の場合は両方 nil になる
// (利用できないデプロイではエクスポートの状態を一切表示しない)。
type GetExportShowOutput struct {
	LatestExport          *model.Export
	LatestSucceededExport *model.Export
	Available             bool
}

// Execute authorizes the signed-in user against the target profile and reads
// the profile's export state.
//
// Authorization runs before the availability check so that a profile the user
// does not own is refused the same way whether or not the deployment can run
// exports. When exports are unavailable the export rows are not read at all:
// the page shows only that the feature is unavailable, so their values could
// not change what it renders.
//
// [Ja] Execute はログイン中ユーザーの対象プロフィールに対する認可を行い、
// プロフィールのエクスポート状態を読む。
//
// 認可を利用可否の判定より先に行うのは、ユーザーが所有していないプロフィールを、
// そのデプロイがエクスポートを実行できるかどうかに関わらず同じ形で拒否するため。
// エクスポートが利用できない場合は export 行を読まない。画面は機能が利用できない
// ことだけを表示するため、その値は描画内容を変えられないからである。
func (uc *GetExportShowUsecase) Execute(ctx context.Context, input GetExportShowInput) (*GetExportShowOutput, error) {
	if err := uc.authorize(ctx, input); err != nil {
		return nil, err
	}

	if !uc.storageReady {
		return &GetExportShowOutput{Available: false}, nil
	}

	latest, err := uc.exportRepo.FindLatestByProfileID(ctx, input.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("最新のエクスポートの取得に失敗: %w", err)
	}

	latestSucceeded, err := uc.exportRepo.FindLatestSucceededByProfileID(ctx, input.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("最新の成功したエクスポートの取得に失敗: %w", err)
	}

	return &GetExportShowOutput{
		LatestExport:          latest,
		LatestSucceededExport: latestSucceeded,
		Available:             true,
	}, nil
}

// authorize checks that the signed-in user currently owns the target profile.
// The session's actor is not the basis for this: an actor row records who
// requested something, while the right to see a profile's exports comes from
// the ownership the user holds right now. Until group-owned profiles exist,
// that ownership is the user_profiles association.
//
// A profile the user does not own is refused as not found, so the response
// cannot be used to tell an existing profile from a missing one.
//
// [Ja] authorize はログイン中ユーザーが対象プロフィールを現在所有していることを
// 確認する。セッションの actor は根拠にしない。actor 行は誰が何を申請したかの記録で
// あり、プロフィールのエクスポートを見る権利は、そのユーザーが今持っている所有関係
// から生じるため。グループ所有プロフィールの導入前は、その所有関係は user_profiles の
// 関連付けである。
//
// 所有していないプロフィールは not found として拒否し、応答から既存のプロフィールと
// 存在しないプロフィールを区別できないようにする。
func (uc *GetExportShowUsecase) authorize(ctx context.Context, input GetExportShowInput) error {
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
