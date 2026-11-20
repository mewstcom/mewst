package repository_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestExportRepository_Create(t *testing.T) {
	t.Parallel()

	// 制約違反のケースはトランザクションをabortさせ共有トランザクションを
	// 汚染するため、各サブテストは自分のトランザクションを開く。

	t.Run("queuedエクスポートを作成できる", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		export, err := repo.Create(ctx, repository.CreateExportInput{
			ProfileID: profileID,
			ActorID:   actorID,
		})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}

		if export.ProfileID != profileID {
			t.Errorf("export.ProfileID = %v、期待値 = %v", export.ProfileID, profileID)
		}
		if export.ActorID != actorID {
			t.Errorf("export.ActorID = %v、期待値 = %v", export.ActorID, actorID)
		}
		if export.Status != model.ExportStatusQueued {
			t.Errorf("export.Status = %v、期待値 = %v", export.Status, model.ExportStatusQueued)
		}
		if export.AttemptCount != 0 {
			t.Errorf("export.AttemptCount = %d、期待値 = 0", export.AttemptCount)
		}
		if export.ObjectKey != nil {
			t.Errorf("export.ObjectKey = %v、期待値 = nil", *export.ObjectKey)
		}
		if export.StartedAt != nil {
			t.Errorf("export.StartedAt = %v、期待値 = nil", *export.StartedAt)
		}
		if export.FinishedAt != nil {
			t.Errorf("export.FinishedAt = %v、期待値 = nil", *export.FinishedAt)
		}
		if export.CreatedAt.IsZero() {
			t.Error("export.CreatedAt = ゼロ値、非ゼロ値を期待")
		}
	})

	t.Run("同一プロフィールで進行中エクスポートは1件だけ作成できる", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		if _, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID}); err != nil {
			t.Fatalf("1件目のCreate()のエラー = %v", err)
		}

		// activeなstatusに対する部分ユニークインデックスは、実際の並行実行
		// でも1件だけが成功することを保証する。ここでは同一セッションでの2件目の
		// Createで検証し、インデックスが即座に拒否する。拒否は
		// ErrActiveExportExistsとして報告され、呼び出し側はドライバーのエラーの形も
		// インデックス名も知らずにこれを識別できる。
		_, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID})
		if err == nil {
			t.Fatal("2件目のCreate() は失敗するべきだがnilが返った")
		}
		if !errors.Is(err, repository.ErrActiveExportExists) {
			t.Errorf("ErrActiveExportExistsを期待したが: %v", err)
		}
	})

	t.Run("actorとprofileが不一致の行を拒否する", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		ownerA := testutil.NewProfileOwner(t, tx)
		actorA := ownerA.ActorID
		ownerB := testutil.NewProfileOwner(t, tx)
		profileB := ownerB.ProfileID
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		// actorAはプロフィールAに属するため、プロフィールBと組み合わせると
		// 複合 (actor_id, profile_id) 外部キーに拒否される。
		_, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileB, ActorID: actorA})
		if err == nil {
			t.Fatal("不一致のCreate() は失敗するべきだがnilが返った")
		}
		if !strings.Contains(err.Error(), "exports_actor_profile_fkey") {
			t.Errorf("exports_actor_profile_fkey違反を期待したが: %v", err)
		}
	})

	t.Run("プロフィール削除開始後は作成しない", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		if _, err := tx.Exec(
			"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
			uuid.UUID(profileID),
		); err != nil {
			t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
		}

		export, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID})
		if err != nil {
			t.Fatalf("Create()のエラー = %v、期待値 = nil", err)
		}
		if export != nil {
			t.Errorf("Create()のexport = %v、期待値 = nil", export)
		}
	})
}

func TestExportRepository_Create_UsesSingleStatementSnapshot(t *testing.T) {
	t.Parallel()

	db := testutil.GetTestDB()
	ctx := context.Background()

	setupTx, err := db.Begin()
	if err != nil {
		t.Fatalf("セットアップ用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = setupTx.Rollback() }()

	userID := testutil.NewUserBuilder(t, setupTx).Build()
	profileID := testutil.NewProfileBuilder(t, setupTx).Build()
	actorID := testutil.NewActorBuilder(t, setupTx).
		WithUserID(userID).
		WithProfileID(profileID).
		Build()
	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, setupTx).Build()
	visiblePostID := testutil.NewPostBuilder(t, setupTx).
		WithProfileID(profileID).
		WithOauthApplicationID(oauthApplicationID).
		WithContent("申請前にcommit済み").
		Build()
	if err := setupTx.Commit(); err != nil {
		t.Fatalf("前提データのコミットに失敗: %v", err)
	}

	// このテストは通常のtransaction内repositoryテストと異なり行をcommit
	// する。参照先のactor/profile/user/applicationより先に子行を削除する。
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM exports WHERE profile_id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM posts WHERE profile_id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM oauth_applications WHERE id = $1", uuid.UUID(oauthApplicationID))
		_, _ = db.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(actorID))
		_, _ = db.Exec("DELETE FROM profiles WHERE id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", uuid.UUID(userID))
	})

	// 別接続でCreateより先にtransactionと書き込みを開始するが、投稿の
	// commitはCreateのcommit後まで保留する。申請時点のsnapshotは、その
	// transactionが後から可視になってもこの投稿を取り込んではならない。
	lateTx, err := db.Begin()
	if err != nil {
		t.Fatalf("後発投稿用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = lateTx.Rollback() }()
	testutil.NewPostBuilder(t, lateTx).
		WithProfileID(profileID).
		WithOauthApplicationID(oauthApplicationID).
		WithContent("申請後にcommit").
		Build()

	exportTx, err := db.Begin()
	if err != nil {
		t.Fatalf("export用トランザクションの開始に失敗: %v", err)
	}
	defer func() { _ = exportTx.Rollback() }()
	export, err := repository.NewExportRepository(query.New(exportTx)).Create(ctx, repository.CreateExportInput{
		ProfileID: profileID,
		ActorID:   actorID,
	})
	if err != nil {
		t.Fatalf("Create()のエラー = %v", err)
	}
	if err := exportTx.Commit(); err != nil {
		t.Fatalf("exportのコミットに失敗: %v", err)
	}

	exportPostRepo := repository.NewExportPostRepository(query.New(db))
	months, err := exportPostRepo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
		ExportID: export.ID,
		Location: time.UTC,
	})
	if err != nil {
		t.Fatalf("ListMonthsByExportID()のエラー = %v", err)
	}
	if len(months) != 1 {
		t.Fatalf("len(months) = %d、期待値 = 1", len(months))
	}

	if err := lateTx.Commit(); err != nil {
		t.Fatalf("後発投稿のコミットに失敗: %v", err)
	}

	posts, _, err := exportPostRepo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
		ExportID: export.ID,
		Month:    months[0],
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListByExportIDInRange()のエラー = %v", err)
	}
	assertPostIDs(t, posts, []model.PostID{visiblePostID})
	if posts[0].Content != "申請前にcommit済み" {
		t.Errorf("posts[0].Content = %q、期待値 = %q", posts[0].Content, "申請前にcommit済み")
	}
}

func TestExportRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

	t.Run("IDを指定してエクスポートを取得できる", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		id := testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusQueued).
			Build()

		got, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if got == nil {
			t.Fatal("FindByID() = nil、エクスポートを期待")
		}
		if got.ID != id {
			t.Errorf("got.ID = %v、期待値 = %v", got.ID, id)
		}
		if got.ProfileID != profileID {
			t.Errorf("got.ProfileID = %v、期待値 = %v", got.ProfileID, profileID)
		}
		if got.ActorID != actorID {
			t.Errorf("got.ActorID = %v、期待値 = %v", got.ActorID, actorID)
		}
	})

	t.Run("存在しないIDはnilを返す", func(t *testing.T) {
		got, err := repo.FindByID(ctx, model.ExportID(uuid.New()))
		if err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		}
		if got != nil {
			t.Errorf("FindByID() = %v、期待値 = nil", got)
		}
	})
}

func TestExportRepository_FindLatestByProfileID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

	t.Run("作成日時が最も新しいエクスポートを返す", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(base).
			Build()
		newerID := testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusQueued).
			WithCreatedAt(base.Add(time.Hour)).
			Build()

		got, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got == nil {
			t.Fatal("FindLatestByProfileID() = nil、新しいほうのエクスポートを期待")
		}
		if got.ID != newerID {
			t.Errorf("got.ID = %v、期待値 = %v", got.ID, newerID)
		}
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusQueued)
		}
	})

	t.Run("エクスポートが無ければnilを返す", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID := owner.ProfileID

		got, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got != nil {
			t.Errorf("FindLatestByProfileID() = %v、期待値 = nil", got)
		}
	})

	t.Run("他プロフィールのエクスポートは返さない", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID := owner.ProfileID
		otherOwner := testutil.NewProfileOwner(t, tx)
		otherProfileID, otherActorID := otherOwner.ProfileID, otherOwner.ActorID
		testutil.NewExportBuilder(t, tx).
			WithProfileID(otherProfileID).WithActorID(otherActorID).
			WithStatus(model.ExportStatusQueued).
			Build()

		got, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got != nil {
			t.Errorf("FindLatestByProfileID() = %v、期待値 = nil", got)
		}
	})
}

func TestExportRepository_FindLatestSucceededByProfileID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

	t.Run("最新のsucceededを返し、より新しいqueuedは無視する", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusSucceeded).
			WithCreatedAt(base).
			Build()
		newerSucceededID := testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusSucceeded).
			WithObjectKey("exports/profile/newer.zip").
			WithCreatedAt(base.Add(time.Hour)).
			Build()
		// さらに新しいqueuedエクスポートが最新のsucceededを隠してはならない。
		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusQueued).
			WithCreatedAt(base.Add(2 * time.Hour)).
			Build()

		got, err := repo.FindLatestSucceededByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestSucceededByProfileID()のエラー = %v", err)
		}
		if got == nil {
			t.Fatal("FindLatestSucceededByProfileID() = nil、新しいほうのsucceededのエクスポートを期待")
		}
		if got.ID != newerSucceededID {
			t.Errorf("got.ID = %v、期待値 = %v", got.ID, newerSucceededID)
		}
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusSucceeded)
		}
		if got.ObjectKey == nil || *got.ObjectKey != "exports/profile/newer.zip" {
			t.Errorf("got.ObjectKey = %v、期待値 = exports/profile/newer.zip", got.ObjectKey)
		}
	})

	t.Run("succeededが無ければnilを返す", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusQueued).
			Build()

		got, err := repo.FindLatestSucceededByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestSucceededByProfileID()のエラー = %v", err)
		}
		if got != nil {
			t.Errorf("FindLatestSucceededByProfileID() = %v、期待値 = nil", got)
		}
	})
}

func TestExportRepository_StateTransitions(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	queries := testutil.QueriesWithTx(tx)
	repo := repository.NewExportRepository(queries)
	notificationRepo := repository.NewExportCompletionNotificationRepository(queries)

	// createQueuedは新しい対象を作り、そのprofileの作成直後のqueued
	// エクスポートを返す。各サブテストが独立したプロフィールから開始する。
	createQueued := func(t *testing.T) *model.Export {
		t.Helper()
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		export, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID})
		if err != nil {
			t.Fatalf("Create()のエラー = %v", err)
		}
		return export
	}
	findLatest := func(t *testing.T, profileID model.ProfileID) *model.Export {
		t.Helper()
		export, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if export == nil {
			t.Fatal("FindLatestByProfileID() = nil、エクスポートを期待")
		}
		return export
	}

	t.Run("MarkStartedはqueuedをstartedにしattempt_countを増やす", func(t *testing.T) {
		export := createQueued(t)

		started, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt)
		if err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		if started == nil {
			t.Fatal("MarkStarted() = nil、startedのエクスポートを期待")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d、期待値 = 1", got.AttemptCount)
		}
		if got.StartedAt == nil {
			t.Error("got.StartedAt = nil、非nilを期待")
		}

		// 返る行は、同じ試行が次の遷移へ提示するトークンであるため、文が書き
		// 込む前の状態ではなく書き込んだ後の状態である必要がある。
		if !started.UpdatedAt.Equal(got.UpdatedAt) {
			t.Errorf("started.UpdatedAt = %v、期待値 = %v", started.UpdatedAt, got.UpdatedAt)
		}
		if started.AttemptCount != got.AttemptCount {
			t.Errorf("started.AttemptCount = %d、期待値 = %d", started.AttemptCount, got.AttemptCount)
		}
	})

	t.Run("MarkStartedはstartedの再入でもattempt_countを増やす", func(t *testing.T) {
		export := createQueued(t)

		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("1回目のMarkStarted()のエラー = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		// Riverのリトライはすでにstartedの行に再入する。ガードはこれを許可し、
		// attempt countは増え続ける。
		retried, err := repo.MarkStarted(ctx, export.ID, started.UpdatedAt)
		if err != nil {
			t.Fatalf("2回目のMarkStarted()のエラー = %v", err)
		}
		if retried == nil {
			t.Fatal("2回目のMarkStarted() = nil、startedのエクスポートを期待")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 2 {
			t.Errorf("got.AttemptCount = %d、期待値 = 2", got.AttemptCount)
		}
	})

	t.Run("MarkSucceededはstartedをsucceededにしobject_keyを設定する", func(t *testing.T) {
		export := createQueued(t)
		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		const objectKey = "exports/profile/export.zip"
		updated, err := repo.MarkSucceeded(ctx, export.ID, objectKey, started.UpdatedAt)
		if err != nil {
			t.Fatalf("MarkSucceeded()のエラー = %v", err)
		}
		if !updated {
			t.Fatal("MarkSucceeded() = false、期待値 = true")
		}

		got, err := repo.FindLatestSucceededByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestSucceededByProfileID()のエラー = %v", err)
		}
		if got == nil {
			t.Fatal("FindLatestSucceededByProfileID() = nil、succeededのエクスポートを期待")
		}
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusSucceeded)
		}
		if got.ObjectKey == nil || *got.ObjectKey != objectKey {
			t.Errorf("got.ObjectKey = %v、期待値 = %v", got.ObjectKey, objectKey)
		}
		if got.FinishedAt == nil {
			t.Fatal("got.FinishedAt = nil、非nilを期待")
		}
		notification, err := notificationRepo.FindByExportID(ctx, export.ID)
		if err != nil {
			t.Fatalf("FindByExportID()のエラー = %v", err)
		}
		if notification == nil {
			t.Fatal("FindByExportID() = nil、送信待ちの通知を期待")
		}
		if notification.ActorID != export.ActorID {
			t.Errorf("notification.ActorID = %v、期待値 = %v", notification.ActorID, export.ActorID)
		}
		var expectedRecipient string
		if err := tx.QueryRow(
			"SELECT users.email FROM users JOIN actors ON actors.user_id = users.id WHERE actors.id = $1",
			uuid.UUID(export.ActorID),
		).Scan(&expectedRecipient); err != nil {
			t.Fatalf("通知先メールアドレスの取得に失敗: %v", err)
		}
		if notification.RecipientEmail != expectedRecipient {
			t.Errorf("notification.RecipientEmail = %q、期待値 = %q", notification.RecipientEmail, expectedRecipient)
		}
		if notification.Locale != "ja" {
			t.Errorf("notification.Locale = %q、期待値 = %q", notification.Locale, "ja")
		}

		if _, err := tx.Exec(
			"UPDATE users SET email = $1, locale = $2 WHERE id = (SELECT user_id FROM actors WHERE id = $3)",
			"changed-after-success@example.com",
			"en",
			uuid.UUID(export.ActorID),
		); err != nil {
			t.Fatalf("成功後のユーザー更新に失敗: %v", err)
		}
		snapshot, err := notificationRepo.FindByExportID(ctx, export.ID)
		if err != nil {
			t.Fatalf("ユーザー更新後のFindByExportID()のエラー = %v", err)
		}
		if snapshot == nil {
			t.Fatal("ユーザー更新後のFindByExportID() = nil、通知を期待")
		}
		if snapshot.RecipientEmail != expectedRecipient || snapshot.Locale != "ja" {
			t.Errorf(
				"ユーザー更新後の通知 = (%q, %q)、期待値 = 固定化した値 (%q, %q)",
				snapshot.RecipientEmail,
				snapshot.Locale,
				expectedRecipient,
				"ja",
			)
		}
		if !notification.CreatedAt.Equal(*got.FinishedAt) {
			t.Errorf("notification.CreatedAt = %v、期待値 = %v", notification.CreatedAt, *got.FinishedAt)
		}
	})

	t.Run("MarkFailedはstartedをfailedにする", func(t *testing.T) {
		export := createQueued(t)
		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		updated, err := repo.MarkFailed(ctx, export.ID, started.UpdatedAt)
		if err != nil {
			t.Fatalf("MarkFailed()のエラー = %v", err)
		}
		if !updated {
			t.Fatal("MarkFailed() = false、期待値 = true")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusFailed)
		}
		if got.FinishedAt == nil {
			t.Error("got.FinishedAt = nil、非nilを期待")
		}
		if got.ObjectKey != nil {
			t.Errorf("got.ObjectKey = %v、期待値 = nil", *got.ObjectKey)
		}
	})

	t.Run("Requeueはstartedをqueuedに戻しstarted_atをクリアしてattempt_countを保持する", func(t *testing.T) {
		export := createQueued(t)
		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("MarkStarted()のエラー = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		updated, err := repo.Requeue(ctx, export.ID, started.UpdatedAt)
		if err != nil {
			t.Fatalf("Requeue()のエラー = %v", err)
		}
		if !updated {
			t.Fatal("Requeue() = false、期待値 = true")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusQueued)
		}
		if got.StartedAt != nil {
			t.Errorf("got.StartedAt = %v、期待値 = nil", *got.StartedAt)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d、期待値 = 1 (差し戻しをまたいで維持)", got.AttemptCount)
		}
	})

	t.Run("古いupdated_atの状態遷移は新しい試行を変更しない", func(t *testing.T) {
		export := createQueued(t)

		attemptA, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt)
		if err != nil || attemptA == nil {
			t.Fatalf("試行AのMarkStarted() = (%v, %v)、期待値 = (startedのエクスポート, nil)", attemptA, err)
		}

		if updated, err := repo.Requeue(ctx, export.ID, attemptA.UpdatedAt); err != nil || !updated {
			t.Fatalf("試行AのRequeue() = (%v, %v)、期待値 = (true, nil)", updated, err)
		}
		requeued := findLatest(t, export.ProfileID)

		attemptB, err := repo.MarkStarted(ctx, export.ID, requeued.UpdatedAt)
		if err != nil || attemptB == nil {
			t.Fatalf("試行BのMarkStarted() = (%v, %v)、期待値 = (startedのエクスポート, nil)", attemptB, err)
		}
		if !attemptB.UpdatedAt.After(attemptA.UpdatedAt) {
			t.Fatalf("試行BのUpdatedAt = %v、期待値 = %vより後", attemptB.UpdatedAt, attemptA.UpdatedAt)
		}

		if stale, err := repo.MarkStarted(ctx, export.ID, attemptA.UpdatedAt); err != nil || stale != nil {
			t.Errorf("古いトークンでのMarkStarted() = (%v, %v)、期待値 = (nil, nil)", stale, err)
		}
		if updated, err := repo.MarkSucceeded(ctx, export.ID, "exports/profile/stale.zip", attemptA.UpdatedAt); err != nil || updated {
			t.Errorf("古いトークンでのMarkSucceeded() = (%v, %v)、期待値 = (false, nil)", updated, err)
		}
		if updated, err := repo.MarkFailed(ctx, export.ID, attemptA.UpdatedAt); err != nil || updated {
			t.Errorf("古いトークンでのMarkFailed() = (%v, %v)、期待値 = (false, nil)", updated, err)
		}
		if updated, err := repo.Requeue(ctx, export.ID, attemptA.UpdatedAt); err != nil || updated {
			t.Errorf("古いトークンでのRequeue() = (%v, %v)、期待値 = (false, nil)", updated, err)
		}

		got := findLatest(t, export.ProfileID)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v、期待値 = %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 2 {
			t.Errorf("got.AttemptCount = %d、期待値 = 2", got.AttemptCount)
		}
		if !got.UpdatedAt.Equal(attemptB.UpdatedAt) {
			t.Errorf("got.UpdatedAt = %v、期待値 = %v", got.UpdatedAt, attemptB.UpdatedAt)
		}
		if got.ObjectKey != nil {
			t.Errorf("got.ObjectKey = %v、期待値 = nil", *got.ObjectKey)
		}
	})

	t.Run("状態が合わない条件付き更新はfalseを返し行を変えない", func(t *testing.T) {
		// MarkSucceeded / MarkFailed / Requeueはstartedを要求し、MarkStartedは
		// 終端状態の行を対象外にする。不一致は0件更新となり、メソッドはエクスポートに
		// 触れずfalseを返す。
		queued := createQueued(t)

		if updated, err := repo.MarkSucceeded(ctx, queued.ID, "exports/profile/export.zip", queued.UpdatedAt); err != nil || updated {
			t.Errorf("queuedに対するMarkSucceeded() = (%v, %v)、期待値 = (false, nil)", updated, err)
		}
		if updated, err := repo.MarkFailed(ctx, queued.ID, queued.UpdatedAt); err != nil || updated {
			t.Errorf("queuedに対するMarkFailed() = (%v, %v)、期待値 = (false, nil)", updated, err)
		}
		if updated, err := repo.Requeue(ctx, queued.ID, queued.UpdatedAt); err != nil || updated {
			t.Errorf("queuedに対するRequeue() = (%v, %v)、期待値 = (false, nil)", updated, err)
		}
		got, err := repo.FindLatestByProfileID(ctx, queued.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if got.Status != model.ExportStatusQueued || got.AttemptCount != 0 {
			t.Errorf("queuedのエクスポートが変更された: status=%v attempt_count=%d、期待値 = queued/0", got.Status, got.AttemptCount)
		}

		// MarkStartedは終端状態 (succeeded) の行を対象外にする。
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusSucceeded).
			Build()
		succeeded, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
		}
		if started, err := repo.MarkStarted(ctx, succeeded.ID, succeeded.UpdatedAt); err != nil || started != nil {
			t.Errorf("succeededに対するMarkStarted() = (%v, %v)、期待値 = (nil, nil)", started, err)
		}
	})

	t.Run("終端状態への遷移は申請時の投稿snapshotを破棄する", func(t *testing.T) {
		// snapshotは再試行が初回と同じ入力を読むためだけに存在し、最新の成功は
		// 期限なしで保持される。終端遷移の一部として破棄することが、エクスポートした
		// 投稿を永久に二重保存しないための仕組みになっている。
		postRepo := repository.NewExportPostRepository(testutil.QueriesWithTx(tx))
		countSnapshot := func(t *testing.T, exportID model.ExportID) int64 {
			t.Helper()
			months, err := postRepo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
				ExportID: exportID,
				Location: time.UTC,
			})
			if err != nil {
				t.Fatalf("ListMonthsByExportID()のエラー = %v", err)
			}
			var total int64
			for _, month := range months {
				total += month.PostCount
			}
			return total
		}

		for _, tc := range []struct {
			name string
			mark func(t *testing.T, export *model.Export) bool
		}{
			{
				name: "MarkSucceeded",
				mark: func(t *testing.T, export *model.Export) bool {
					t.Helper()
					updated, err := repo.MarkSucceeded(ctx, export.ID, "exports/profile/export.zip", export.UpdatedAt)
					if err != nil {
						t.Fatalf("MarkSucceeded()のエラー = %v", err)
					}
					return updated
				},
			},
			{
				name: "MarkFailed",
				mark: func(t *testing.T, export *model.Export) bool {
					t.Helper()
					updated, err := repo.MarkFailed(ctx, export.ID, export.UpdatedAt)
					if err != nil {
						t.Fatalf("MarkFailed()のエラー = %v", err)
					}
					return updated
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				owner := testutil.NewProfileOwner(t, tx)
				profileID, actorID := owner.ProfileID, owner.ActorID
				testutil.NewPostBuilder(t, tx).
					WithProfileID(profileID).
					WithOauthApplicationID(testutil.NewOauthApplicationBuilder(t, tx).Build()).
					WithPublishedAt(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)).
					Build()

				export, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID})
				if err != nil {
					t.Fatalf("Create()のエラー = %v", err)
				}
				if got := countSnapshot(t, export.ID); got != 1 {
					t.Fatalf("作成直後のsnapshot件数 = %d、期待値 = 1", got)
				}

				started, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt)
				if err != nil || started == nil {
					t.Fatalf("MarkStarted() = (%v, %v)、期待値 = (startedのエクスポート, nil)", started, err)
				}
				// リトライはsnapshotを読み直すため、エクスポートが終端状態に
				// 達するまでは残っている必要がある。
				if got := countSnapshot(t, export.ID); got != 1 {
					t.Fatalf("startedでのsnapshot件数 = %d、期待値 = 1", got)
				}

				if !tc.mark(t, started) {
					t.Fatal("終端状態への遷移でガードが一致しなかった")
				}
				if got := countSnapshot(t, export.ID); got != 0 {
					t.Errorf("終端状態でのsnapshot件数 = %d、期待値 = 0", got)
				}
			})
		}
	})
}

// buildExportは指定した対象・status・created_atのエクスポートを挿入し、
// そのIDを返す。
func buildExport(t *testing.T, tx *sql.Tx, profileID model.ProfileID, actorID model.ActorID, status model.ExportStatus, createdAt time.Time) model.ExportID {
	t.Helper()
	return testutil.NewExportBuilder(t, tx).
		WithProfileID(profileID).WithActorID(actorID).
		WithStatus(status).
		WithCreatedAt(createdAt).
		Build()
}

// newExportWithStatusは新しい対象を作り、その対象に指定status・created_at
// のエクスポートを作る。各行が独立したプロフィールに載るため、statusがqueued /
// startedのときのactive部分ユニークインデックスを満たせる。
func newExportWithStatus(t *testing.T, tx *sql.Tx, status model.ExportStatus, createdAt time.Time) model.ExportID {
	t.Helper()
	owner := testutil.NewProfileOwner(t, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID
	return buildExport(t, tx, profileID, actorID, status, createdAt)
}

// sortExportIDsはPostgreSQLのuuidのバイト順でIDをソートする。
// クエリで使うORDER BY idと一致させる。
func sortExportIDs(ids []model.ExportID) {
	slices.SortFunc(ids, func(a, b model.ExportID) int {
		ua, ub := uuid.UUID(a), uuid.UUID(b)
		return bytes.Compare(ua[:], ub[:])
	})
}

// assertExportIDsInOrderはgotのエクスポートが指定順のwant IDと
// ちょうど一致しなければ失敗する。
func assertExportIDsInOrder(t *testing.T, got []*model.Export, want ...model.ExportID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("返されたエクスポートの件数 = %d、期待値 = %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("got[%d].ID = %v、期待値 = %v", i, got[i].ID, id)
		}
	}
}

// assertNextCursorはcursorが指定の位置を指していなければ失敗する。すなわち
// repositoryがその一覧メソッドの並び順カラムからcursorを組み立てたことを検証する。
func assertNextCursor(t *testing.T, cursor *repository.ExportRecoveryCursor, wantTimestamp time.Time, wantID model.ExportID) {
	t.Helper()
	if cursor == nil {
		t.Fatal("次のカーソル = nil、満杯のページに対するカーソルを期待")
	}
	if !cursor.Timestamp.Equal(wantTimestamp) {
		t.Errorf("cursor.Timestamp = %v、期待値 = %v", cursor.Timestamp, wantTimestamp)
	}
	if cursor.ID != wantID {
		t.Errorf("cursor.ID = %v、期待値 = %v", cursor.ID, wantID)
	}
}

// assertNoNextCursorはcursorがnilでなければ失敗する。nilは一覧メソッドが
// 「ページが埋まらず走査は終わり」と伝える手段。
func assertNoNextCursor(t *testing.T, cursor *repository.ExportRecoveryCursor) {
	t.Helper()
	if cursor != nil {
		t.Errorf("次のカーソル = %+v、期待値 = nil", cursor)
	}
}

// exportIDsはエクスポートのIDを現在の順序のまま返す。
func exportIDs(exports []*model.Export) []model.ExportID {
	ids := make([]model.ExportID, len(exports))
	for i, export := range exports {
		ids[i] = export.ID
	}
	return ids
}

// unboundedTestPageSizeは共有テストDBより意図的に十分大きくし、テストが
// 全件を見たい結果をpage sizeで切り捨てないようにする。
const unboundedTestPageSize int32 = 1 << 30

// exportSetはテストが作ったエクスポートを記録し、他パッケージが共有テスト
// DBにコミットした行を検証から無視できるようにする。回復クエリはprofileで
// スコープされないためコミット済みの全行が見えるが、SetupTxを使うテストが所有
// するのは自分のトランザクションで挿入した行だけ。
type exportSet struct {
	t   *testing.T
	tx  *sql.Tx
	ids map[model.ExportID]bool
}

func newExportSet(t *testing.T, tx *sql.Tx) *exportSet {
	t.Helper()
	return &exportSet{t: t, tx: tx, ids: make(map[model.ExportID]bool)}
}

// addは新しい対象に指定status・created_atのエクスポートを作り、この
// テストが所有する行として記録する。
func (s *exportSet) add(status model.ExportStatus, createdAt time.Time) model.ExportID {
	s.t.Helper()
	id := newExportWithStatus(s.t, s.tx, status, createdAt)
	s.ids[id] = true
	return id
}

// ownedはgotのうちこのテストが作ったエクスポートを、クエリが返した順序を
// 保って返す。
func (s *exportSet) owned(got []*model.Export) []*model.Export {
	s.t.Helper()
	filtered := make([]*model.Export, 0, len(got))
	for _, e := range got {
		if s.ids[e.ID] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func TestExportRepository_ListStaleQueued(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTxRepeatableRead(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
	threshold := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	set := newExportSet(t, tx)

	// thresholdより前に作成されたqueued: 生成ジョブが投入されなかったため、
	// 再投入の対象として返る必要がある。
	oldest := set.add(model.ExportStatusQueued, threshold.Add(-2*time.Hour))
	tied := []model.ExportID{
		set.add(model.ExportStatusQueued, threshold.Add(-time.Hour)),
		set.add(model.ExportStatusQueued, threshold.Add(-time.Hour)),
	}
	sortExportIDs(tied)

	// ちょうどthresholdに作成: 厳密な < により除外される (境界時刻)。
	set.add(model.ExportStatusQueued, threshold)
	// thresholdより後に作成: まだ猶予期間内で除外される。
	set.add(model.ExportStatusQueued, threshold.Add(time.Hour))
	// queuedでない行は経過時間に関わらず対象外。
	set.add(model.ExportStatusStarted, threshold.Add(-time.Hour))
	set.add(model.ExportStatusSucceeded, threshold.Add(-time.Hour))
	set.add(model.ExportStatusFailed, threshold.Add(-time.Hour))

	// 削除が始まったプロフィールのqueued: 生成はマーカーで止まるため、この行を
	// 再投入しても、行に触れずに戻るジョブが生まれるだけである。行を消すのは親削除。
	deletingOwner := testutil.NewProfileOwner(t, tx)
	deletingProfile, deletingActor := deletingOwner.ProfileID, deletingOwner.ActorID
	deleting := buildExport(t, tx, deletingProfile, deletingActor, model.ExportStatusQueued, threshold.Add(-time.Hour))
	if _, err := tx.Exec(
		"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
		uuid.UUID(deletingProfile),
	); err != nil {
		t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
	}

	got, next, err := repo.ListStaleQueued(ctx, threshold, nil, unboundedTestPageSize)
	if err != nil {
		t.Fatalf("ListStaleQueued()のエラー = %v", err)
	}
	assertExportIDsInOrder(t, set.owned(got), oldest, tied[0], tied[1])
	assertNoNextCursor(t, next)

	// 絞り込み前の結果に対して検証する。set.ownedはクエリが返したかどうかに
	// 関わらずこの行を落とすため、除外されたことを示せないため。
	if slices.Contains(exportIDs(got), deleting) {
		t.Errorf("削除開始済みプロフィールのqueuedが返っている: %v", deleting)
	}

	// 他テストの行を除外せず、グローバルな結果をページングする。これにより
	// 正確なページ上限と、repositoryが返したcursorがその先へ進むことの両方を
	// 検証する。
	firstPage, cursor, err := repo.ListStaleQueued(ctx, threshold, nil, 2)
	if err != nil {
		t.Fatalf("1ページ目のListStaleQueued()のエラー = %v", err)
	}
	assertExportIDsInOrder(t, firstPage, exportIDs(got[:2])...)
	assertNextCursor(t, cursor, firstPage[len(firstPage)-1].CreatedAt, firstPage[len(firstPage)-1].ID)

	secondPage, _, err := repo.ListStaleQueued(ctx, threshold, cursor, 2)
	if err != nil {
		t.Fatalf("2ページ目のListStaleQueued()のエラー = %v", err)
	}
	assertExportIDsInOrder(t, secondPage, exportIDs(got[2:min(4, len(got))])...)
}

func TestExportRepository_ListStaleStarted(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTxRepeatableRead(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
	threshold := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	set := newExportSet(t, tx)

	// 試行がthresholdより前に始まったstarted: Workerがタイムアウトを
	// 超えたため、回復の対象として返る必要がある。
	oldest := set.add(model.ExportStatusStarted, threshold.Add(-2*time.Hour))
	tied := []model.ExportID{
		set.add(model.ExportStatusStarted, threshold.Add(-time.Hour)),
		set.add(model.ExportStatusStarted, threshold.Add(-time.Hour)),
	}
	sortExportIDs(tied)

	// started_atがちょうどthresholdの行は厳密な < により除外される (境界)。
	set.add(model.ExportStatusStarted, threshold)
	set.add(model.ExportStatusStarted, threshold.Add(time.Hour))
	// startedでない行は経過時間に関わらず対象外。
	set.add(model.ExportStatusQueued, threshold.Add(-time.Hour))
	set.add(model.ExportStatusSucceeded, threshold.Add(-time.Hour))

	// 削除が始まったプロフィールのstarted: 生成はマーカーで止まるため、この行を
	// 差し戻しても、行に触れずに戻るジョブが生まれるだけである。行を消すのは親削除。
	deletingOwner := testutil.NewProfileOwner(t, tx)
	deletingProfile, deletingActor := deletingOwner.ProfileID, deletingOwner.ActorID
	deleting := buildExport(t, tx, deletingProfile, deletingActor, model.ExportStatusStarted, threshold.Add(-time.Hour))
	if _, err := tx.Exec(
		"UPDATE profiles SET export_deletion_started_at = NOW() WHERE id = $1",
		uuid.UUID(deletingProfile),
	); err != nil {
		t.Fatalf("プロフィール削除開始の記録に失敗: %v", err)
	}

	got, next, err := repo.ListStaleStarted(ctx, threshold, nil, unboundedTestPageSize)
	if err != nil {
		t.Fatalf("ListStaleStarted()のエラー = %v", err)
	}
	assertExportIDsInOrder(t, set.owned(got), oldest, tied[0], tied[1])
	assertNoNextCursor(t, next)

	// 絞り込み前の結果に対して検証する。set.ownedはクエリが返したかどうかに
	// 関わらずこの行を落とすため、除外されたことを示せないため。
	if slices.Contains(exportIDs(got), deleting) {
		t.Errorf("削除開始済みプロフィールのstartedが返っている: %v", deleting)
	}

	// グローバルな結果をページングし、正確な上限と、cursorがcreated_atでは
	// なくstarted_atに従うことを検証する。
	firstPage, cursor, err := repo.ListStaleStarted(ctx, threshold, nil, 2)
	if err != nil {
		t.Fatalf("1ページ目のListStaleStarted()のエラー = %v", err)
	}
	assertExportIDsInOrder(t, firstPage, exportIDs(got[:2])...)
	assertNextCursor(t, cursor, *firstPage[len(firstPage)-1].StartedAt, firstPage[len(firstPage)-1].ID)

	secondPage, _, err := repo.ListStaleStarted(ctx, threshold, cursor, 2)
	if err != nil {
		t.Fatalf("2ページ目のListStaleStarted()のエラー = %v", err)
	}
	assertExportIDsInOrder(t, secondPage, exportIDs(got[2:min(4, len(got))])...)
}

func TestExportRepository_ListOldSucceededByProfileID(t *testing.T) {
	t.Parallel()

	t.Run("最新以外のsucceededを返し、最新を除外する", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		old1 := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		old2 := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(time.Hour))
		latest := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(2*time.Hour))

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID()のエラー = %v", err)
		}
		assertExportIDsInOrder(t, got, old1, old2)
		for _, e := range got {
			if e.ID == latest {
				t.Errorf("最新のsucceeded %vが削除対象に選ばれた", latest)
			}
		}
	})

	t.Run("同一created_atではidでtie-breakし、最新 (最大id) を除外する", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		sameTime := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		ids := make([]model.ExportID, 0, 3)
		for range 3 {
			ids = append(ids, buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, sameTime))
		}
		sortExportIDs(ids)
		latest := ids[len(ids)-1]

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID()のエラー = %v", err)
		}
		assertExportIDsInOrder(t, got, ids[:len(ids)-1]...)
		for _, e := range got {
			if e.ID == latest {
				t.Errorf("最新のsucceeded %v (idが最大) が削除対象に選ばれた", latest)
			}
		}
	})

	t.Run("非succeededと他プロフィールを候補にしない", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		old := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(2*time.Hour)) // latest
		// 同一プロフィールのfailed / queuedはcleanup候補にしてはならない。
		buildExport(t, tx, profileID, actorID, model.ExportStatusFailed, base.Add(time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusQueued, base.Add(3*time.Hour))

		// 別プロフィールのsucceededが混入してはならない。
		otherOwner := testutil.NewProfileOwner(t, tx)
		otherProfile, otherActor := otherOwner.ProfileID, otherOwner.ActorID
		buildExport(t, tx, otherProfile, otherActor, model.ExportStatusSucceeded, base)
		buildExport(t, tx, otherProfile, otherActor, model.ExportStatusSucceeded, base.Add(time.Hour))

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID()のエラー = %v", err)
		}
		assertExportIDsInOrder(t, got, old)
	})

	t.Run("succeededが1件だけなら空を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID

		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, time.Now())

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID()のエラー = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("返されたエクスポートの件数 = %d、期待値 = 0", len(got))
		}
	})

	t.Run("page sizeを超える候補は切り捨て、古い順の先頭を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		// 古いsucceeded 3件 + 最新1件。page size 2で動いたcleanupは最も
		// 古い2件を取り、残りはそれらが削除された次回の実行で拾われる。
		old1 := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		old2 := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(2*time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(3*time.Hour))

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, 2)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID()のエラー = %v", err)
		}
		assertExportIDsInOrder(t, got, old1, old2)
	})
}

func TestExportRepository_ListProfileIDsWithOldSucceeded(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTxRepeatableRead(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
	base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

	// succeeded 2件: 掃除すべき古い行があるため、このプロフィールは返る。
	ownerTwo := testutil.NewProfileOwner(t, tx)
	withTwo, actorTwo := ownerTwo.ProfileID, ownerTwo.ActorID
	buildExport(t, tx, withTwo, actorTwo, model.ExportStatusSucceeded, base)
	buildExport(t, tx, withTwo, actorTwo, model.ExportStatusSucceeded, base.Add(time.Hour))

	// 2件目の対象プロフィールにより、page size 1の次ページが実在することを
	// 保証する。
	ownerTwoSecond := testutil.NewProfileOwner(t, tx)
	withTwoSecond, actorTwoSecond := ownerTwoSecond.ProfileID, ownerTwoSecond.ActorID
	buildExport(t, tx, withTwoSecond, actorTwoSecond, model.ExportStatusSucceeded, base)
	buildExport(t, tx, withTwoSecond, actorTwoSecond, model.ExportStatusSucceeded, base.Add(time.Hour))

	// succeeded 1件: 古い行が無いため、このプロフィールは返らない。
	ownerOne := testutil.NewProfileOwner(t, tx)
	withOne, actorOne := ownerOne.ProfileID, ownerOne.ActorID
	buildExport(t, tx, withOne, actorOne, model.ExportStatusSucceeded, base)

	// succeeded 1件 + failed 1件: succeededは依然1件のため返らない。
	ownerFailed := testutil.NewProfileOwner(t, tx)
	withFailed, actorFailed := ownerFailed.ProfileID, ownerFailed.ActorID
	buildExport(t, tx, withFailed, actorFailed, model.ExportStatusSucceeded, base)
	buildExport(t, tx, withFailed, actorFailed, model.ExportStatusFailed, base.Add(time.Hour))

	got, next, err := repo.ListProfileIDsWithOldSucceeded(ctx, nil, unboundedTestPageSize)
	if err != nil {
		t.Fatalf("ListProfileIDsWithOldSucceeded()のエラー = %v", err)
	}
	if next != nil {
		t.Errorf("次のカーソル = %v、期待値 = nil", next)
	}
	gotSet := make(map[model.ProfileID]bool, len(got))
	for _, id := range got {
		gotSet[id] = true
	}
	if !gotSet[withTwo] {
		t.Errorf("succeededを2件持つプロフィール%vが結果に無い", withTwo)
	}
	if !gotSet[withTwoSecond] {
		t.Errorf("succeededを2件持つ2つ目のプロフィール%vが結果に無い", withTwoSecond)
	}
	if gotSet[withOne] {
		t.Errorf("succeededを1件だけ持つプロフィール%vが返された", withOne)
	}
	if gotSet[withFailed] {
		t.Errorf("succeededとfailedを1件ずつ持つプロフィール%vが返された", withFailed)
	}

	if !slices.IsSortedFunc(got, func(a, b model.ProfileID) int {
		ua, ub := uuid.UUID(a), uuid.UUID(b)
		return bytes.Compare(ua[:], ub[:])
	}) {
		t.Errorf("ListProfileIDsWithOldSucceeded()の結果が並んでいない: %v", got)
	}

	// グローバルな結果をページングし、ページサイズ上限によって2ページ目
	// 以降のプロフィールが取り残されないことを検証する。
	firstPage, cursor, err := repo.ListProfileIDsWithOldSucceeded(ctx, nil, 1)
	if err != nil {
		t.Fatalf("1ページ目のListProfileIDsWithOldSucceeded()のエラー = %v", err)
	}
	if len(firstPage) != 1 || firstPage[0] != got[0] {
		t.Fatalf("1ページ目 = %v、期待値 = [%v]", firstPage, got[0])
	}
	if cursor == nil || *cursor != firstPage[0] {
		t.Fatalf("次のカーソル = %v、期待値 = %v", cursor, firstPage[0])
	}

	secondPage, _, err := repo.ListProfileIDsWithOldSucceeded(ctx, cursor, 1)
	if err != nil {
		t.Fatalf("2ページ目のListProfileIDsWithOldSucceeded()のエラー = %v", err)
	}
	if len(secondPage) != 1 || secondPage[0] != got[1] {
		t.Fatalf("2ページ目 = %v、期待値 = [%v]", secondPage, got[1])
	}
}

func TestExportRepository_FindIDsRetainingObject(t *testing.T) {
	t.Parallel()

	t.Run("オブジェクトを保持しているIDだけを返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		queued := newExportWithStatus(t, tx, model.ExportStatusQueued, time.Now())
		started := newExportWithStatus(t, tx, model.ExportStatusStarted, time.Now())
		succeeded := newExportWithStatus(t, tx, model.ExportStatusSucceeded, time.Now())
		// failedのエクスポートはオブジェクトを手放している。手放すのが終端遷移で
		// あるため、このIDでまだ保存されているオブジェクトは、この行が生かしている
		// ものではなく、掃除が回収すべき孤児である。
		failed := newExportWithStatus(t, tx, model.ExportStatusFailed, time.Now())
		missing := model.ExportID(uuid.New())

		got, err := repo.FindIDsRetainingObject(ctx, []model.ExportID{queued, started, succeeded, failed, missing})
		if err != nil {
			t.Fatalf("FindIDsRetainingObject()のエラー = %v", err)
		}
		gotSet := make(map[model.ExportID]bool, len(got))
		for _, id := range got {
			gotSet[id] = true
		}
		if len(gotSet) != 3 || !gotSet[queued] || !gotSet[started] || !gotSet[succeeded] {
			t.Errorf("FindIDsRetainingObject() = %v、期待値 = {%v, %v, %v}", got, queued, started, succeeded)
		}
		if gotSet[failed] {
			t.Errorf("failedのエクスポート%vが保持側に含まれています", failed)
		}
		if gotSet[missing] {
			t.Errorf("存在しないID %vが結果に含まれています", missing)
		}
	})

	t.Run("空入力はnilを返しクエリを発行しない", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		got, err := repo.FindIDsRetainingObject(ctx, nil)
		if err != nil {
			t.Fatalf("FindIDsRetainingObject()のエラー = %v", err)
		}
		if got != nil {
			t.Errorf("FindIDsRetainingObject(nil) = %v、期待値 = nil", got)
		}
	})
}

// TestExportRepository_ListByProfileIDは、プロフィールの削除処理が削除対象と
// して受け取るものを固定する。全statusが含まれる必要がある。試行はそれを記録する
// 遷移より先にアップロードするため、queued / started / failedのエクスポートも
// オブジェクトを持ちうる。それらを除いた一覧は、削除されたプロフィールに、どこからも
// 指されないアーカイブを残させることになる。
func TestExportRepository_ListByProfileID(t *testing.T) {
	t.Parallel()

	t.Run("statusを問わず古い順に返し、他プロフィールを含めない", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		succeeded := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		failed := buildExport(t, tx, profileID, actorID, model.ExportStatusFailed, base.Add(time.Hour))
		queued := buildExport(t, tx, profileID, actorID, model.ExportStatusQueued, base.Add(2*time.Hour))

		otherOwner := testutil.NewProfileOwner(t, tx)
		otherProfile, otherActor := otherOwner.ProfileID, otherOwner.ActorID
		buildExport(t, tx, otherProfile, otherActor, model.ExportStatusSucceeded, base)

		got, err := repo.ListByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListByProfileID()のエラー = %v", err)
		}
		assertExportIDsInOrder(t, got, succeeded, failed, queued)
	})

	t.Run("page sizeを超える行は切り捨て、古い順の先頭を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		// 呼び出し側は処理した行を削除するため、残りは次のクエリの先頭に現れ、
		// cursorは不要になる。
		first := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		second := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(2*time.Hour))

		got, err := repo.ListByProfileID(ctx, profileID, 2)
		if err != nil {
			t.Fatalf("ListByProfileID()のエラー = %v", err)
		}
		assertExportIDsInOrder(t, got, first, second)
	})

	t.Run("エクスポートが無ければ空を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID := owner.ProfileID

		got, err := repo.ListByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListByProfileID()のエラー = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("返されたエクスポートの件数 = %d、期待値 = 0", len(got))
		}
	})
}

func TestExportRepository_Delete(t *testing.T) {
	t.Parallel()

	statuses := []model.ExportStatus{
		model.ExportStatusQueued,
		model.ExportStatusStarted,
		model.ExportStatusSucceeded,
		model.ExportStatusFailed,
	}
	for _, status := range statuses {
		t.Run(status.String(), func(t *testing.T) {
			_, tx := testutil.SetupTx(t)
			ctx := context.Background()
			repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
			owner := testutil.NewProfileOwner(t, tx)
			profileID, actorID := owner.ProfileID, owner.ActorID

			id := buildExport(t, tx, profileID, actorID, status, time.Now())

			deleted, err := repo.Delete(ctx, id)
			if err != nil {
				t.Fatalf("Delete()のエラー = %v", err)
			}
			if !deleted {
				t.Fatal("Delete() = false、期待値 = true")
			}

			remaining, err := repo.FindLatestByProfileID(ctx, profileID)
			if err != nil {
				t.Fatalf("FindLatestByProfileID()のエラー = %v", err)
			}
			if remaining != nil {
				t.Errorf("Delete後もエクスポートが残っている: %v", remaining.ID)
			}
		})
	}

	t.Run("存在しない行の削除はfalseを返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		deleted, err := repo.Delete(ctx, model.ExportID(uuid.New()))
		if err != nil {
			t.Fatalf("Delete()のエラー = %v", err)
		}
		if deleted {
			t.Error("存在しない行に対するDelete() = true、期待値 = false")
		}
	})
}

// TestExportRepository_DeleteFailedByProfileIDは、Createのtransactionが
// 何を片付けるかを固定する。対象はそのプロフィール自身のfailedなエクスポートだけ
// である。
func TestExportRepository_DeleteFailedByProfileID(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	t.Run("failedの行だけを削除する", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID

		failedID := buildExport(t, tx, profileID, actorID, model.ExportStatusFailed, base)
		succeededID := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(time.Hour))

		deleted, err := repo.DeleteFailedByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("DeleteFailedByProfileID()のエラー = %v", err)
		}
		if deleted != 1 {
			t.Errorf("DeleteFailedByProfileID() = %d、期待値 = 1", deleted)
		}

		if got, err := repo.FindByID(ctx, failedID); err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		} else if got != nil {
			t.Errorf("failedのエクスポートが残っている: %v", got.ID)
		}
		if got, err := repo.FindByID(ctx, succeededID); err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		} else if got == nil {
			t.Error("succeededのエクスポートが削除されている")
		}
	})

	// プロフィールの実行中エクスポートはactiveな部分ユニークインデックスが
	// 守っており、ここで消すと2件目のエクスポートを並行して開始できてしまう。
	for _, status := range []model.ExportStatus{model.ExportStatusQueued, model.ExportStatusStarted} {
		t.Run(status.String()+" の行は削除しない", func(t *testing.T) {
			_, tx := testutil.SetupTx(t)
			ctx := context.Background()
			repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
			owner := testutil.NewProfileOwner(t, tx)
			profileID, actorID := owner.ProfileID, owner.ActorID

			id := buildExport(t, tx, profileID, actorID, status, base)

			deleted, err := repo.DeleteFailedByProfileID(ctx, profileID)
			if err != nil {
				t.Fatalf("DeleteFailedByProfileID()のエラー = %v", err)
			}
			if deleted != 0 {
				t.Errorf("DeleteFailedByProfileID() = %d、期待値 = 0", deleted)
			}

			if got, err := repo.FindByID(ctx, id); err != nil {
				t.Fatalf("FindByID()のエラー = %v", err)
			} else if got == nil {
				t.Errorf("%sのエクスポートが削除されている", status)
			}
		})
	}

	t.Run("他プロフィールのfailedは削除しない", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		target := testutil.NewProfileOwner(t, tx)
		other := testutil.NewProfileOwner(t, tx)

		otherFailedID := buildExport(t, tx, other.ProfileID, other.ActorID, model.ExportStatusFailed, base)

		deleted, err := repo.DeleteFailedByProfileID(ctx, target.ProfileID)
		if err != nil {
			t.Fatalf("DeleteFailedByProfileID()のエラー = %v", err)
		}
		if deleted != 0 {
			t.Errorf("DeleteFailedByProfileID() = %d、期待値 = 0", deleted)
		}

		if got, err := repo.FindByID(ctx, otherFailedID); err != nil {
			t.Fatalf("FindByID()のエラー = %v", err)
		} else if got == nil {
			t.Error("他プロフィールのfailedが削除されている")
		}
	})

	// failedなエクスポートのsnapshotは純粋な従属データであり、
	// export_postsのCASCADEは行と一緒にこれを消さなければならない。
	t.Run("export_postsもCASCADEで消える", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)

		failedID := buildExport(t, tx, owner.ProfileID, owner.ActorID, model.ExportStatusFailed, base)
		if _, err := tx.Exec(
			"INSERT INTO export_posts (export_id, post_id, content, published_at) VALUES ($1, $2, $3, $4)",
			uuid.UUID(failedID), uuid.New(), "hello", base,
		); err != nil {
			t.Fatalf("export_postsの作成に失敗: %v", err)
		}

		if _, err := repo.DeleteFailedByProfileID(ctx, owner.ProfileID); err != nil {
			t.Fatalf("DeleteFailedByProfileID()のエラー = %v", err)
		}

		var remaining int
		if err := tx.QueryRow(
			"SELECT COUNT(*) FROM export_posts WHERE export_id = $1",
			uuid.UUID(failedID),
		).Scan(&remaining); err != nil {
			t.Fatalf("export_postsの件数取得に失敗: %v", err)
		}
		if remaining != 0 {
			t.Errorf("export_postsが%d件残っている、期待値 = 0", remaining)
		}
	})
}
