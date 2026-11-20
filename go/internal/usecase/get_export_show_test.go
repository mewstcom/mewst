package usecase_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

func newGetExportShowUsecase(t *testing.T, tx *sql.Tx, storageReady bool) *usecase.GetExportShowUsecase {
	t.Helper()

	queries := testutil.QueriesWithTx(tx)
	return usecase.NewGetExportShowUsecase(
		repository.NewUserProfileRepository(queries),
		repository.NewExportRepository(queries),
		storageReady,
	)
}

// TestGetExportShowUsecase_Executeは、エクスポート画面がプロフィールの
// エクスポートについて何を知らされるか、および誰が問い合わせられるかを固定する。
func TestGetExportShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	t.Run("エクスポートが無いプロフィールでは両方nilを返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		uc := newGetExportShowUsecase(t, tx, true)

		output, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.LatestExport != nil {
			t.Errorf("LatestExport = %v、期待値 = nil", output.LatestExport)
		}
		if output.LatestSucceededExport != nil {
			t.Errorf("LatestSucceededExport = %v、期待値 = nil", output.LatestSucceededExport)
		}
		if !output.Available {
			t.Error("Available = false、期待値 = true")
		}
	})

	// より新しい申請は、以前の申請が作ったアーカイブを隠してはならない。画面は
	// 新しいエクスポートの実行中や、それが諦めた後も、以前のzipを提供し続ける。
	t.Run("進行中のエクスポートがあっても以前の成功を併せて返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		uc := newGetExportShowUsecase(t, tx, true)

		succeededID := testutil.NewExportBuilder(t, tx).
			WithProfileID(owner.ProfileID).
			WithActorID(owner.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(base).
			Build()
		queuedID := testutil.NewExportBuilder(t, tx).
			WithProfileID(owner.ProfileID).
			WithActorID(owner.ActorID).
			WithStatus(model.ExportStatusQueued).
			WithCreatedAt(base.Add(time.Hour)).
			Build()

		output, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.LatestExport == nil || output.LatestExport.ID != queuedID {
			t.Errorf("LatestExport = %v、期待値 = %v", output.LatestExport, queuedID)
		}
		if output.LatestSucceededExport == nil || output.LatestSucceededExport.ID != succeededID {
			t.Errorf("LatestSucceededExport = %v、期待値 = %v", output.LatestSucceededExport, succeededID)
		}
	})

	t.Run("最新が成功なら両方その行を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		uc := newGetExportShowUsecase(t, tx, true)

		testutil.NewExportBuilder(t, tx).
			WithProfileID(owner.ProfileID).
			WithActorID(owner.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(base).
			Build()
		latestID := testutil.NewExportBuilder(t, tx).
			WithProfileID(owner.ProfileID).
			WithActorID(owner.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(base.Add(time.Hour)).
			Build()

		output, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.LatestExport == nil || output.LatestExport.ID != latestID {
			t.Errorf("LatestExport = %v、期待値 = %v", output.LatestExport, latestID)
		}
		if output.LatestSucceededExport == nil || output.LatestSucceededExport.ID != latestID {
			t.Errorf("LatestSucceededExport = %v、期待値 = %v", output.LatestSucceededExport, latestID)
		}
	})

	// 他ユーザーのプロフィールは、存在しないプロフィールと同じ形で拒否する。
	// 応答から、そのプロフィールが存在することや、エクスポートを持つことを知られない
	// ようにするため。
	t.Run("他ユーザーが所有するプロフィールはnot foundとして拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)
		uc := newGetExportShowUsecase(t, tx, true)

		testutil.NewExportBuilder(t, tx).
			WithProfileID(owner.ProfileID).
			WithActorID(owner.ActorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(base).
			Build()

		output, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    other.UserID,
			ProfileID: owner.ProfileID,
		})
		if output != nil {
			t.Errorf("出力 = %v、期待値 = nil", output)
		}

		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
		}
		if appErr.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v、期待値 = %v", appErr.Code, model.AppErrCodeResourceNotFound)
		}
	})

	// アクターだけを認可の根拠にはしない。現在の所有者がいないプロフィールは、
	// それを指すアクター行が残っていても拒否する。
	t.Run("所有関係が無いプロフィールはnot foundとして拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		userID := testutil.NewUserBuilder(t, tx).Build()
		// 所有種別を明示し、所有されたプロフィールとの差がuser_profiles行の
		// 欠落だけになるようにする。本番でこの状態になるのは、ユーザー所有の
		// プロフィールから関連付けが失われた場合であるため。
		profileID := testutil.NewProfileBuilder(t, tx).
			WithOwnerType(model.ProfileOwnerTypeUser).
			Build()
		testutil.NewActorBuilder(t, tx).
			WithUserID(userID).
			WithProfileID(profileID).
			Build()
		uc := newGetExportShowUsecase(t, tx, true)

		_, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    userID,
			ProfileID: profileID,
		})

		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
		}
		if appErr.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v、期待値 = %v", appErr.Code, model.AppErrCodeResourceNotFound)
		}
	})

	// オブジェクトストレージが無ければ画面はエクスポートを開始することも提供する
	// こともできないため、機能を利用不可として報告し、export行は読まない。
	t.Run("ストレージ未設定なら利用不可を返しエクスポートを読まない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)

		// nilのexport Repositoryにより読み取り境界を観測可能にする。この
		// ケースが成功するのは、利用不可の分岐がどちらのexport取得よりも先に
		// returnする場合だけである。
		queries := testutil.QueriesWithTx(tx)
		uc := usecase.NewGetExportShowUsecase(
			repository.NewUserProfileRepository(queries),
			nil,
			false,
		)

		output, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if output.Available {
			t.Error("Available = true、期待値 = false")
		}
		if output.LatestExport != nil {
			t.Errorf("LatestExport = %v、期待値 = nil", output.LatestExport)
		}
		if output.LatestSucceededExport != nil {
			t.Errorf("LatestSucceededExport = %v、期待値 = nil", output.LatestSucceededExport)
		}
	})

	// 認可を先に行うため、利用できないデプロイでも他人のプロフィールには
	// 利用不可の状態を返さず拒否する。
	t.Run("ストレージ未設定でも他ユーザーのプロフィールは拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)
		uc := newGetExportShowUsecase(t, tx, false)

		_, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    other.UserID,
			ProfileID: owner.ProfileID,
		})

		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute()のエラー = %v、*model.AppErrorを期待", err)
		}
		if appErr.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v、期待値 = %v", appErr.Code, model.AppErrCodeResourceNotFound)
		}
	})
}
