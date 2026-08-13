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

// TestGetExportShowUsecase_Execute pins what the export page is told about a
// profile's exports, and who is allowed to ask.
//
// [Ja] TestGetExportShowUsecase_Execute は、エクスポート画面がプロフィールの
// エクスポートについて何を知らされるか、および誰が問い合わせられるかを固定する。
func TestGetExportShowUsecase_Execute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	t.Run("エクスポートが無いプロフィールでは両方 nil を返す", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)
		uc := newGetExportShowUsecase(t, tx, true)

		output, err := uc.Execute(ctx, usecase.GetExportShowInput{
			UserID:    owner.UserID,
			ProfileID: owner.ProfileID,
		})
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if output.LatestExport != nil {
			t.Errorf("LatestExport = %v, want nil", output.LatestExport)
		}
		if output.LatestSucceededExport != nil {
			t.Errorf("LatestSucceededExport = %v, want nil", output.LatestSucceededExport)
		}
		if !output.Available {
			t.Error("Available = false, want true")
		}
	})

	// A newer request must not hide the archive an earlier one produced: the
	// page keeps offering the previous zip while the new export runs or after it
	// gives up.
	//
	// [Ja] より新しい申請は、以前の申請が作ったアーカイブを隠してはならない。画面は
	// 新しいエクスポートの実行中や、それが諦めた後も、以前の zip を提供し続ける。
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
			t.Fatalf("Execute() error = %v", err)
		}

		if output.LatestExport == nil || output.LatestExport.ID != queuedID {
			t.Errorf("LatestExport = %v, want %v", output.LatestExport, queuedID)
		}
		if output.LatestSucceededExport == nil || output.LatestSucceededExport.ID != succeededID {
			t.Errorf("LatestSucceededExport = %v, want %v", output.LatestSucceededExport, succeededID)
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
			t.Fatalf("Execute() error = %v", err)
		}

		if output.LatestExport == nil || output.LatestExport.ID != latestID {
			t.Errorf("LatestExport = %v, want %v", output.LatestExport, latestID)
		}
		if output.LatestSucceededExport == nil || output.LatestSucceededExport.ID != latestID {
			t.Errorf("LatestSucceededExport = %v, want %v", output.LatestSucceededExport, latestID)
		}
	})

	// Another user's profile must be refused the same way a profile that does
	// not exist is, so the response cannot be used to learn that the profile
	// exists or that it has exports.
	//
	// [Ja] 他ユーザーのプロフィールは、存在しないプロフィールと同じ形で拒否する。
	// 応答から、そのプロフィールが存在することや、エクスポートを持つことを知られない
	// ようにするため。
	t.Run("他ユーザーが所有するプロフィールは not found として拒否する", func(t *testing.T) {
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
			t.Errorf("output = %v, want nil", output)
		}

		appErr := model.AsAppError(err)
		if appErr == nil {
			t.Fatalf("Execute() error = %v, want *model.AppError", err)
		}
		if appErr.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", appErr.Code, model.AppErrCodeResourceNotFound)
		}
	})

	// The actor alone is not the basis for authorization: a profile with no
	// current owner is refused even though an actor row still points at it.
	//
	// [Ja] アクターだけを認可の根拠にはしない。現在の所有者がいないプロフィールは、
	// それを指すアクター行が残っていても拒否する。
	t.Run("所有関係が無いプロフィールは not found として拒否する", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		userID := testutil.NewUserBuilder(t, tx).Build()
		// Spell out the owner type so the fixture differs from an owned profile
		// only by the missing user_profiles row, which is the shape production
		// reaches when the association is removed from a user-owned profile.
		//
		// [Ja] 所有種別を明示し、所有されたプロフィールとの差が user_profiles 行の
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
			t.Fatalf("Execute() error = %v, want *model.AppError", err)
		}
		if appErr.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", appErr.Code, model.AppErrCodeResourceNotFound)
		}
	})

	// Without the object storage the page can neither start an export nor serve
	// one, so it reports the feature as unavailable and leaves the export rows
	// unread.
	//
	// [Ja] オブジェクトストレージが無ければ画面はエクスポートを開始することも提供する
	// こともできないため、機能を利用不可として報告し、export 行は読まない。
	t.Run("ストレージ未設定なら利用不可を返しエクスポートを読まない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		owner := testutil.NewProfileOwner(t, tx)

		// A nil export repository makes the read boundary observable: this case
		// succeeds only when the unavailable branch returns before either export
		// lookup.
		//
		// [Ja] nil の export Repository により読み取り境界を観測可能にする。この
		// ケースが成功するのは、利用不可の分岐がどちらの export 取得よりも先に
		// return する場合だけである。
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
			t.Fatalf("Execute() error = %v", err)
		}

		if output.Available {
			t.Error("Available = true, want false")
		}
		if output.LatestExport != nil {
			t.Errorf("LatestExport = %v, want nil", output.LatestExport)
		}
		if output.LatestSucceededExport != nil {
			t.Errorf("LatestSucceededExport = %v, want nil", output.LatestSucceededExport)
		}
	})

	// Authorization runs first, so an unavailable deployment refuses a foreign
	// profile instead of answering it with the unavailable state.
	//
	// [Ja] 認可を先に行うため、利用できないデプロイでも他人のプロフィールには
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
			t.Fatalf("Execute() error = %v, want *model.AppError", err)
		}
		if appErr.Code != model.AppErrCodeResourceNotFound {
			t.Errorf("AppError.Code = %v, want %v", appErr.Code, model.AppErrCodeResourceNotFound)
		}
	})
}
