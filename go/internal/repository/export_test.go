package repository_test

import (
	"bytes"
	"context"
	"database/sql"
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

	// Each sub-test opens its own transaction because the constraint-violation
	// cases abort the transaction, which would poison a shared one.
	//
	// [Ja] 制約違反のケースはトランザクションを abort させ共有トランザクションを
	// 汚染するため、各サブテストは自分のトランザクションを開く。

	t.Run("queued エクスポートを作成できる", func(t *testing.T) {
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
			t.Fatalf("Create() error = %v", err)
		}

		if export.ProfileID != profileID {
			t.Errorf("export.ProfileID = %v, want %v", export.ProfileID, profileID)
		}
		if export.ActorID != actorID {
			t.Errorf("export.ActorID = %v, want %v", export.ActorID, actorID)
		}
		if export.Status != model.ExportStatusQueued {
			t.Errorf("export.Status = %v, want %v", export.Status, model.ExportStatusQueued)
		}
		if export.AttemptCount != 0 {
			t.Errorf("export.AttemptCount = %d, want 0", export.AttemptCount)
		}
		if export.ObjectKey != nil {
			t.Errorf("export.ObjectKey = %v, want nil", *export.ObjectKey)
		}
		if export.StartedAt != nil {
			t.Errorf("export.StartedAt = %v, want nil", *export.StartedAt)
		}
		if export.FinishedAt != nil {
			t.Errorf("export.FinishedAt = %v, want nil", *export.FinishedAt)
		}
		if export.CreatedAt.IsZero() {
			t.Error("export.CreatedAt should not be zero")
		}
	})

	t.Run("同一プロフィールで進行中エクスポートは 1 件だけ作成できる", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		if _, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID}); err != nil {
			t.Fatalf("1 件目の Create() error = %v", err)
		}

		// The partial unique index on active statuses guarantees only one wins
		// even under real concurrency; it is exercised here by a second create in
		// the same session, which the index rejects immediately.
		//
		// [Ja] active な status に対する部分ユニークインデックスは、実際の並行実行
		// でも 1 件だけが成功することを保証する。ここでは同一セッションでの 2 件目の
		// Create で検証し、インデックスが即座に拒否する。
		_, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID})
		if err == nil {
			t.Fatal("2 件目の Create() は失敗するべきだが nil が返った")
		}
		if !strings.Contains(err.Error(), "index_exports_on_profile_id_where_active") {
			t.Errorf("index_exports_on_profile_id_where_active 違反を期待したが: %v", err)
		}
	})

	t.Run("actor と profile が不一致の行を拒否する", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		ownerA := testutil.NewProfileOwner(t, tx)
		actorA := ownerA.ActorID
		ownerB := testutil.NewProfileOwner(t, tx)
		profileB := ownerB.ProfileID
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		// actorA belongs to profile A, so pairing it with profile B must be
		// rejected by the composite (actor_id, profile_id) foreign key.
		//
		// [Ja] actorA はプロフィール A に属するため、プロフィール B と組み合わせると
		// 複合 (actor_id, profile_id) 外部キーに拒否される。
		_, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileB, ActorID: actorA})
		if err == nil {
			t.Fatal("不一致の Create() は失敗するべきだが nil が返った")
		}
		if !strings.Contains(err.Error(), "exports_actor_profile_fkey") {
			t.Errorf("exports_actor_profile_fkey 違反を期待したが: %v", err)
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
			t.Fatalf("Create() error = %v, want nil", err)
		}
		if export != nil {
			t.Errorf("Create() export = %v, want nil", export)
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

	// This test commits rows, unlike the transaction-scoped repository tests.
	// Remove children before their referenced actor/profile/user/application.
	//
	// [Ja] このテストは通常の transaction 内 repository テストと異なり行を commit
	// する。参照先の actor/profile/user/application より先に子行を削除する。
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM exports WHERE profile_id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM posts WHERE profile_id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM oauth_applications WHERE id = $1", uuid.UUID(oauthApplicationID))
		_, _ = db.Exec("DELETE FROM actors WHERE id = $1", uuid.UUID(actorID))
		_, _ = db.Exec("DELETE FROM profiles WHERE id = $1", uuid.UUID(profileID))
		_, _ = db.Exec("DELETE FROM users WHERE id = $1", uuid.UUID(userID))
	})

	// Start and write on another connection before Create, but leave the post
	// uncommitted until after Create commits. A request-time snapshot must not
	// gain this post when that later transaction becomes visible.
	//
	// [Ja] 別接続で Create より先に transaction と書き込みを開始するが、投稿の
	// commit は Create の commit 後まで保留する。申請時点の snapshot は、その
	// transaction が後から可視になってもこの投稿を取り込んではならない。
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
		t.Fatalf("Create() error = %v", err)
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
		t.Fatalf("ListMonthsByExportID() error = %v", err)
	}
	if len(months) != 1 {
		t.Fatalf("len(months) = %d, want 1", len(months))
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
		t.Fatalf("ListByExportIDInRange() error = %v", err)
	}
	assertPostIDs(t, posts, []model.PostID{visiblePostID})
	if posts[0].Content != "申請前にcommit済み" {
		t.Errorf("posts[0].Content = %q, want %q", posts[0].Content, "申請前にcommit済み")
	}
}

func TestExportRepository_FindByID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

	t.Run("ID を指定してエクスポートを取得できる", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		id := testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusQueued).
			Build()

		got, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if got == nil {
			t.Fatal("FindByID() = nil, want the export")
		}
		if got.ID != id {
			t.Errorf("got.ID = %v, want %v", got.ID, id)
		}
		if got.ProfileID != profileID {
			t.Errorf("got.ProfileID = %v, want %v", got.ProfileID, profileID)
		}
		if got.ActorID != actorID {
			t.Errorf("got.ActorID = %v, want %v", got.ActorID, actorID)
		}
	})

	t.Run("存在しない ID は nil を返す", func(t *testing.T) {
		got, err := repo.FindByID(ctx, model.ExportID(uuid.New()))
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if got != nil {
			t.Errorf("FindByID() = %v, want nil", got)
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
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got == nil {
			t.Fatal("FindLatestByProfileID() = nil, want the newer export")
		}
		if got.ID != newerID {
			t.Errorf("got.ID = %v, want %v", got.ID, newerID)
		}
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusQueued)
		}
	})

	t.Run("エクスポートが無ければ nil を返す", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID := owner.ProfileID

		got, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got != nil {
			t.Errorf("FindLatestByProfileID() = %v, want nil", got)
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
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got != nil {
			t.Errorf("FindLatestByProfileID() = %v, want nil", got)
		}
	})
}

func TestExportRepository_FindLatestSucceededByProfileID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

	t.Run("最新の succeeded を返し、より新しい queued は無視する", func(t *testing.T) {
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
		// A still newer queued export must not shadow the latest succeeded one.
		//
		// [Ja] さらに新しい queued エクスポートが最新の succeeded を隠してはならない。
		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusQueued).
			WithCreatedAt(base.Add(2 * time.Hour)).
			Build()

		got, err := repo.FindLatestSucceededByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestSucceededByProfileID() error = %v", err)
		}
		if got == nil {
			t.Fatal("FindLatestSucceededByProfileID() = nil, want the newer succeeded export")
		}
		if got.ID != newerSucceededID {
			t.Errorf("got.ID = %v, want %v", got.ID, newerSucceededID)
		}
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusSucceeded)
		}
		if got.ObjectKey == nil || *got.ObjectKey != "exports/profile/newer.zip" {
			t.Errorf("got.ObjectKey = %v, want exports/profile/newer.zip", got.ObjectKey)
		}
	})

	t.Run("succeeded が無ければ nil を返す", func(t *testing.T) {
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusQueued).
			Build()

		got, err := repo.FindLatestSucceededByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestSucceededByProfileID() error = %v", err)
		}
		if got != nil {
			t.Errorf("FindLatestSucceededByProfileID() = %v, want nil", got)
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

	// createQueued creates a fresh target and returns a newly created queued
	// export for it, so each sub-test starts from an independent profile.
	//
	// [Ja] createQueued は新しい対象を作り、その profile の作成直後の queued
	// エクスポートを返す。各サブテストが独立したプロフィールから開始する。
	createQueued := func(t *testing.T) *model.Export {
		t.Helper()
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		export, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		return export
	}
	findLatest := func(t *testing.T, profileID model.ProfileID) *model.Export {
		t.Helper()
		export, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if export == nil {
			t.Fatal("FindLatestByProfileID() = nil, want an export")
		}
		return export
	}

	t.Run("MarkStarted は queued を started にし attempt_count を増やす", func(t *testing.T) {
		export := createQueued(t)

		started, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt)
		if err != nil {
			t.Fatalf("MarkStarted() error = %v", err)
		}
		if started == nil {
			t.Fatal("MarkStarted() = nil, want the started export")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d, want 1", got.AttemptCount)
		}
		if got.StartedAt == nil {
			t.Error("got.StartedAt should not be nil")
		}

		// The returned row is the token the same attempt presents to the next
		// transition, so it has to be the state the statement wrote rather than
		// the state before it.
		//
		// [Ja] 返る行は、同じ試行が次の遷移へ提示するトークンであるため、文が書き
		// 込む前の状態ではなく書き込んだ後の状態である必要がある。
		if !started.UpdatedAt.Equal(got.UpdatedAt) {
			t.Errorf("started.UpdatedAt = %v, want %v", started.UpdatedAt, got.UpdatedAt)
		}
		if started.AttemptCount != got.AttemptCount {
			t.Errorf("started.AttemptCount = %d, want %d", started.AttemptCount, got.AttemptCount)
		}
	})

	t.Run("MarkStarted は started の再入でも attempt_count を増やす", func(t *testing.T) {
		export := createQueued(t)

		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("1 回目の MarkStarted() error = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		// A River retry re-enters an already started row; the guard allows it and
		// the attempt count keeps climbing.
		//
		// [Ja] River のリトライはすでに started の行に再入する。ガードはこれを許可し、
		// attempt count は増え続ける。
		retried, err := repo.MarkStarted(ctx, export.ID, started.UpdatedAt)
		if err != nil {
			t.Fatalf("2 回目の MarkStarted() error = %v", err)
		}
		if retried == nil {
			t.Fatal("2 回目の MarkStarted() = nil, want the started export")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 2 {
			t.Errorf("got.AttemptCount = %d, want 2", got.AttemptCount)
		}
	})

	t.Run("MarkSucceeded は started を succeeded にし object_key を設定する", func(t *testing.T) {
		export := createQueued(t)
		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("MarkStarted() error = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		const objectKey = "exports/profile/export.zip"
		updated, err := repo.MarkSucceeded(ctx, export.ID, objectKey, started.UpdatedAt)
		if err != nil {
			t.Fatalf("MarkSucceeded() error = %v", err)
		}
		if !updated {
			t.Fatal("MarkSucceeded() = false, want true")
		}

		got, err := repo.FindLatestSucceededByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestSucceededByProfileID() error = %v", err)
		}
		if got == nil {
			t.Fatal("FindLatestSucceededByProfileID() = nil, want the succeeded export")
		}
		if got.Status != model.ExportStatusSucceeded {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusSucceeded)
		}
		if got.ObjectKey == nil || *got.ObjectKey != objectKey {
			t.Errorf("got.ObjectKey = %v, want %v", got.ObjectKey, objectKey)
		}
		if got.FinishedAt == nil {
			t.Fatal("got.FinishedAt should not be nil")
		}
		notification, err := notificationRepo.FindByExportID(ctx, export.ID)
		if err != nil {
			t.Fatalf("FindByExportID() error = %v", err)
		}
		if notification == nil {
			t.Fatal("FindByExportID() = nil, want a pending notification")
		}
		if notification.ActorID != export.ActorID {
			t.Errorf("notification.ActorID = %v, want %v", notification.ActorID, export.ActorID)
		}
		var expectedRecipient string
		if err := tx.QueryRow(
			"SELECT users.email FROM users JOIN actors ON actors.user_id = users.id WHERE actors.id = $1",
			uuid.UUID(export.ActorID),
		).Scan(&expectedRecipient); err != nil {
			t.Fatalf("通知先メールアドレスの取得に失敗: %v", err)
		}
		if notification.RecipientEmail != expectedRecipient {
			t.Errorf("notification.RecipientEmail = %q, want %q", notification.RecipientEmail, expectedRecipient)
		}
		if notification.Locale != "ja" {
			t.Errorf("notification.Locale = %q, want %q", notification.Locale, "ja")
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
			t.Fatalf("FindByExportID() after user update error = %v", err)
		}
		if snapshot == nil {
			t.Fatal("FindByExportID() after user update = nil, want a notification")
		}
		if snapshot.RecipientEmail != expectedRecipient || snapshot.Locale != "ja" {
			t.Errorf(
				"notification after user update = (%q, %q), want snapshotted (%q, %q)",
				snapshot.RecipientEmail,
				snapshot.Locale,
				expectedRecipient,
				"ja",
			)
		}
		if !notification.CreatedAt.Equal(*got.FinishedAt) {
			t.Errorf("notification.CreatedAt = %v, want %v", notification.CreatedAt, *got.FinishedAt)
		}
	})

	t.Run("MarkFailed は started を failed にする", func(t *testing.T) {
		export := createQueued(t)
		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("MarkStarted() error = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		updated, err := repo.MarkFailed(ctx, export.ID, started.UpdatedAt)
		if err != nil {
			t.Fatalf("MarkFailed() error = %v", err)
		}
		if !updated {
			t.Fatal("MarkFailed() = false, want true")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got.Status != model.ExportStatusFailed {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusFailed)
		}
		if got.FinishedAt == nil {
			t.Error("got.FinishedAt should not be nil")
		}
		if got.ObjectKey != nil {
			t.Errorf("got.ObjectKey = %v, want nil", *got.ObjectKey)
		}
	})

	t.Run("Requeue は started を queued に戻し started_at をクリアして attempt_count を保持する", func(t *testing.T) {
		export := createQueued(t)
		if _, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt); err != nil {
			t.Fatalf("MarkStarted() error = %v", err)
		}
		started := findLatest(t, export.ProfileID)

		updated, err := repo.Requeue(ctx, export.ID, started.UpdatedAt)
		if err != nil {
			t.Fatalf("Requeue() error = %v", err)
		}
		if !updated {
			t.Fatal("Requeue() = false, want true")
		}

		got, err := repo.FindLatestByProfileID(ctx, export.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got.Status != model.ExportStatusQueued {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusQueued)
		}
		if got.StartedAt != nil {
			t.Errorf("got.StartedAt = %v, want nil", *got.StartedAt)
		}
		if got.AttemptCount != 1 {
			t.Errorf("got.AttemptCount = %d, want 1 (kept across requeue)", got.AttemptCount)
		}
	})

	t.Run("古い updated_at の状態遷移は新しい試行を変更しない", func(t *testing.T) {
		export := createQueued(t)

		attemptA, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt)
		if err != nil || attemptA == nil {
			t.Fatalf("試行 A の MarkStarted() = (%v, %v), want (the started export, nil)", attemptA, err)
		}

		if updated, err := repo.Requeue(ctx, export.ID, attemptA.UpdatedAt); err != nil || !updated {
			t.Fatalf("試行 A の Requeue() = (%v, %v), want (true, nil)", updated, err)
		}
		requeued := findLatest(t, export.ProfileID)

		attemptB, err := repo.MarkStarted(ctx, export.ID, requeued.UpdatedAt)
		if err != nil || attemptB == nil {
			t.Fatalf("試行 B の MarkStarted() = (%v, %v), want (the started export, nil)", attemptB, err)
		}
		if !attemptB.UpdatedAt.After(attemptA.UpdatedAt) {
			t.Fatalf("試行 B の UpdatedAt = %v, want after %v", attemptB.UpdatedAt, attemptA.UpdatedAt)
		}

		if stale, err := repo.MarkStarted(ctx, export.ID, attemptA.UpdatedAt); err != nil || stale != nil {
			t.Errorf("stale MarkStarted() = (%v, %v), want (nil, nil)", stale, err)
		}
		if updated, err := repo.MarkSucceeded(ctx, export.ID, "exports/profile/stale.zip", attemptA.UpdatedAt); err != nil || updated {
			t.Errorf("stale MarkSucceeded() = (%v, %v), want (false, nil)", updated, err)
		}
		if updated, err := repo.MarkFailed(ctx, export.ID, attemptA.UpdatedAt); err != nil || updated {
			t.Errorf("stale MarkFailed() = (%v, %v), want (false, nil)", updated, err)
		}
		if updated, err := repo.Requeue(ctx, export.ID, attemptA.UpdatedAt); err != nil || updated {
			t.Errorf("stale Requeue() = (%v, %v), want (false, nil)", updated, err)
		}

		got := findLatest(t, export.ProfileID)
		if got.Status != model.ExportStatusStarted {
			t.Errorf("got.Status = %v, want %v", got.Status, model.ExportStatusStarted)
		}
		if got.AttemptCount != 2 {
			t.Errorf("got.AttemptCount = %d, want 2", got.AttemptCount)
		}
		if !got.UpdatedAt.Equal(attemptB.UpdatedAt) {
			t.Errorf("got.UpdatedAt = %v, want %v", got.UpdatedAt, attemptB.UpdatedAt)
		}
		if got.ObjectKey != nil {
			t.Errorf("got.ObjectKey = %v, want nil", *got.ObjectKey)
		}
	})

	t.Run("状態が合わない条件付き更新は false を返し行を変えない", func(t *testing.T) {
		// MarkSucceeded / MarkFailed / Requeue require started; MarkStarted skips
		// terminal rows. Each mismatch updates zero rows, so the method reports
		// false without touching the export.
		//
		// [Ja] MarkSucceeded / MarkFailed / Requeue は started を要求し、MarkStarted は
		// 終端状態の行を対象外にする。不一致は 0 件更新となり、メソッドはエクスポートに
		// 触れず false を返す。
		queued := createQueued(t)

		if updated, err := repo.MarkSucceeded(ctx, queued.ID, "exports/profile/export.zip", queued.UpdatedAt); err != nil || updated {
			t.Errorf("MarkSucceeded() on queued = (%v, %v), want (false, nil)", updated, err)
		}
		if updated, err := repo.MarkFailed(ctx, queued.ID, queued.UpdatedAt); err != nil || updated {
			t.Errorf("MarkFailed() on queued = (%v, %v), want (false, nil)", updated, err)
		}
		if updated, err := repo.Requeue(ctx, queued.ID, queued.UpdatedAt); err != nil || updated {
			t.Errorf("Requeue() on queued = (%v, %v), want (false, nil)", updated, err)
		}
		got, err := repo.FindLatestByProfileID(ctx, queued.ProfileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if got.Status != model.ExportStatusQueued || got.AttemptCount != 0 {
			t.Errorf("queued export was modified: status=%v attempt_count=%d, want queued/0", got.Status, got.AttemptCount)
		}

		// MarkStarted must skip a terminal (succeeded) row.
		//
		// [Ja] MarkStarted は終端状態 (succeeded) の行を対象外にする。
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		testutil.NewExportBuilder(t, tx).
			WithProfileID(profileID).WithActorID(actorID).
			WithStatus(model.ExportStatusSucceeded).
			Build()
		succeeded, err := repo.FindLatestByProfileID(ctx, profileID)
		if err != nil {
			t.Fatalf("FindLatestByProfileID() error = %v", err)
		}
		if started, err := repo.MarkStarted(ctx, succeeded.ID, succeeded.UpdatedAt); err != nil || started != nil {
			t.Errorf("MarkStarted() on succeeded = (%v, %v), want (nil, nil)", started, err)
		}
	})

	t.Run("終端状態への遷移は申請時の投稿 snapshot を破棄する", func(t *testing.T) {
		// The snapshot only exists so that a retried attempt reads the same
		// input as the first one, and the latest success is kept without
		// expiry. Discarding it as part of the terminal transition is what
		// keeps every exported post from being stored twice forever.
		//
		// [Ja] snapshot は再試行が初回と同じ入力を読むためだけに存在し、最新の成功は
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
				t.Fatalf("ListMonthsByExportID() error = %v", err)
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
						t.Fatalf("MarkSucceeded() error = %v", err)
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
						t.Fatalf("MarkFailed() error = %v", err)
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
					t.Fatalf("Create() error = %v", err)
				}
				if got := countSnapshot(t, export.ID); got != 1 {
					t.Fatalf("作成直後の snapshot 件数 = %d, want 1", got)
				}

				started, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt)
				if err != nil || started == nil {
					t.Fatalf("MarkStarted() = (%v, %v), want (the started export, nil)", started, err)
				}
				// A retry re-reads the snapshot, so it must survive until the
				// export reaches a terminal status.
				//
				// [Ja] リトライは snapshot を読み直すため、エクスポートが終端状態に
				// 達するまでは残っている必要がある。
				if got := countSnapshot(t, export.ID); got != 1 {
					t.Fatalf("started での snapshot 件数 = %d, want 1", got)
				}

				if !tc.mark(t, started) {
					t.Fatal("終端状態への遷移でガードが一致しなかった")
				}
				if got := countSnapshot(t, export.ID); got != 0 {
					t.Errorf("終端状態での snapshot 件数 = %d, want 0", got)
				}
			})
		}
	})
}

// buildExport inserts an export with the given status and created_at for the
// given target, returning its ID.
//
// [Ja] buildExport は指定した対象・status・created_at のエクスポートを挿入し、
// その ID を返す。
func buildExport(t *testing.T, tx *sql.Tx, profileID model.ProfileID, actorID model.ActorID, status model.ExportStatus, createdAt time.Time) model.ExportID {
	t.Helper()
	return testutil.NewExportBuilder(t, tx).
		WithProfileID(profileID).WithActorID(actorID).
		WithStatus(status).
		WithCreatedAt(createdAt).
		Build()
}

// newExportWithStatus creates a fresh target and an export with the given
// status and created_at for it, so each row lives on an independent profile
// (required for the active partial-unique index when the status is queued or
// started).
//
// [Ja] newExportWithStatus は新しい対象を作り、その対象に指定 status・created_at
// のエクスポートを作る。各行が独立したプロフィールに載るため、status が queued /
// started のときの active 部分ユニークインデックスを満たせる。
func newExportWithStatus(t *testing.T, tx *sql.Tx, status model.ExportStatus, createdAt time.Time) model.ExportID {
	t.Helper()
	owner := testutil.NewProfileOwner(t, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID
	return buildExport(t, tx, profileID, actorID, status, createdAt)
}

// sortExportIDs sorts IDs in PostgreSQL's uuid byte order, matching the
// ORDER BY id used by the queries.
//
// [Ja] sortExportIDs は PostgreSQL の uuid のバイト順で ID をソートする。
// クエリで使う ORDER BY id と一致させる。
func sortExportIDs(ids []model.ExportID) {
	slices.SortFunc(ids, func(a, b model.ExportID) int {
		ua, ub := uuid.UUID(a), uuid.UUID(b)
		return bytes.Compare(ua[:], ub[:])
	})
}

// assertExportIDsInOrder fails unless the exports in got contain exactly the
// wanted IDs in the given order.
//
// [Ja] assertExportIDsInOrder は got のエクスポートが指定順の want ID と
// ちょうど一致しなければ失敗する。
func assertExportIDsInOrder(t *testing.T, got []*model.Export, want ...model.ExportID) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("returned %d exports, want %d", len(got), len(want))
	}
	for i, id := range want {
		if got[i].ID != id {
			t.Errorf("got[%d].ID = %v, want %v", i, got[i].ID, id)
		}
	}
}

// assertNextCursor fails unless the cursor points at the given position, i.e.
// the repository built it from the ordering column of that list method.
//
// [Ja] assertNextCursor は cursor が指定の位置を指していなければ失敗する。すなわち
// repository がその一覧メソッドの並び順カラムから cursor を組み立てたことを検証する。
func assertNextCursor(t *testing.T, cursor *repository.ExportRecoveryCursor, wantTimestamp time.Time, wantID model.ExportID) {
	t.Helper()
	if cursor == nil {
		t.Fatal("next cursor = nil, want a cursor for the full page")
	}
	if !cursor.Timestamp.Equal(wantTimestamp) {
		t.Errorf("cursor.Timestamp = %v, want %v", cursor.Timestamp, wantTimestamp)
	}
	if cursor.ID != wantID {
		t.Errorf("cursor.ID = %v, want %v", cursor.ID, wantID)
	}
}

// assertNoNextCursor fails unless the cursor is nil, which is how a list method
// reports that the page was not full and the walk is finished.
//
// [Ja] assertNoNextCursor は cursor が nil でなければ失敗する。nil は一覧メソッドが
// 「ページが埋まらず走査は終わり」と伝える手段。
func assertNoNextCursor(t *testing.T, cursor *repository.ExportRecoveryCursor) {
	t.Helper()
	if cursor != nil {
		t.Errorf("next cursor = %+v, want nil", cursor)
	}
}

// exportIDs returns the IDs from exports in their existing order.
//
// [Ja] exportIDs はエクスポートの ID を現在の順序のまま返す。
func exportIDs(exports []*model.Export) []model.ExportID {
	ids := make([]model.ExportID, len(exports))
	for i, export := range exports {
		ids[i] = export.ID
	}
	return ids
}

// unboundedTestPageSize is intentionally much larger than the shared test DB,
// so a page size never truncates a result the test wants in full.
//
// [Ja] unboundedTestPageSize は共有テスト DB より意図的に十分大きくし、テストが
// 全件を見たい結果を page size で切り捨てないようにする。
const unboundedTestPageSize int32 = 1 << 30

// exportSet records the exports a test creates so assertions can ignore rows
// that other packages committed to the shared test database. The recovery
// queries are not scoped to a profile, so they see every committed row, while
// tests using SetupTx only own the rows they inserted in their own
// transaction.
//
// [Ja] exportSet はテストが作ったエクスポートを記録し、他パッケージが共有テスト
// DB にコミットした行を検証から無視できるようにする。回復クエリは profile で
// スコープされないためコミット済みの全行が見えるが、SetupTx を使うテストが所有
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

// add creates an export with the given status and created_at on a fresh target
// and records it as owned by this test.
//
// [Ja] add は新しい対象に指定 status・created_at のエクスポートを作り、この
// テストが所有する行として記録する。
func (s *exportSet) add(status model.ExportStatus, createdAt time.Time) model.ExportID {
	s.t.Helper()
	id := newExportWithStatus(s.t, s.tx, status, createdAt)
	s.ids[id] = true
	return id
}

// owned returns the exports of got that this test created, preserving the
// order the query returned them in.
//
// [Ja] owned は got のうちこのテストが作ったエクスポートを、クエリが返した順序を
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

	// A queued export created before the threshold: its generation job was never
	// inserted, so it must be returned for re-enqueue.
	//
	// [Ja] threshold より前に作成された queued: 生成ジョブが投入されなかったため、
	// 再投入の対象として返る必要がある。
	oldest := set.add(model.ExportStatusQueued, threshold.Add(-2*time.Hour))
	tied := []model.ExportID{
		set.add(model.ExportStatusQueued, threshold.Add(-time.Hour)),
		set.add(model.ExportStatusQueued, threshold.Add(-time.Hour)),
	}
	sortExportIDs(tied)

	// Created exactly at the threshold: the strict < excludes it (boundary time).
	//
	// [Ja] ちょうど threshold に作成: 厳密な < により除外される (境界時刻)。
	set.add(model.ExportStatusQueued, threshold)
	// Created after the threshold: still within grace, excluded.
	//
	// [Ja] threshold より後に作成: まだ猶予期間内で除外される。
	set.add(model.ExportStatusQueued, threshold.Add(time.Hour))
	// Non-queued rows are out of scope regardless of age.
	//
	// [Ja] queued でない行は経過時間に関わらず対象外。
	set.add(model.ExportStatusStarted, threshold.Add(-time.Hour))
	set.add(model.ExportStatusSucceeded, threshold.Add(-time.Hour))
	set.add(model.ExportStatusFailed, threshold.Add(-time.Hour))

	// Queued on a profile whose deletion has started: generation stops at the
	// marker, so re-enqueueing this row would only produce a job that returns
	// without touching it. Profile deletion is what removes the row.
	//
	// [Ja] 削除が始まったプロフィールの queued: 生成はマーカーで止まるため、この行を
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
		t.Fatalf("ListStaleQueued() error = %v", err)
	}
	assertExportIDsInOrder(t, set.owned(got), oldest, tied[0], tied[1])
	assertNoNextCursor(t, next)

	// Asserted against the unfiltered result: set.owned would drop this row
	// whether the query returned it or not, so it cannot show the exclusion.
	//
	// [Ja] 絞り込み前の結果に対して検証する。set.owned はクエリが返したかどうかに
	// 関わらずこの行を落とすため、除外されたことを示せないため。
	if slices.Contains(exportIDs(got), deleting) {
		t.Errorf("削除開始済みプロフィールの queued が返っている: %v", deleting)
	}

	// Page through the raw global result without filtering foreign rows. This
	// proves both the exact page bound and that the cursor the repository
	// returned advances past it.
	//
	// [Ja] 他テストの行を除外せず、グローバルな結果をページングする。これにより
	// 正確なページ上限と、repository が返した cursor がその先へ進むことの両方を
	// 検証する。
	firstPage, cursor, err := repo.ListStaleQueued(ctx, threshold, nil, 2)
	if err != nil {
		t.Fatalf("ListStaleQueued() first page error = %v", err)
	}
	assertExportIDsInOrder(t, firstPage, exportIDs(got[:2])...)
	assertNextCursor(t, cursor, firstPage[len(firstPage)-1].CreatedAt, firstPage[len(firstPage)-1].ID)

	secondPage, _, err := repo.ListStaleQueued(ctx, threshold, cursor, 2)
	if err != nil {
		t.Fatalf("ListStaleQueued() second page error = %v", err)
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

	// A started export whose attempt began before the threshold: the worker
	// exceeded its timeout, so it must be returned for recovery.
	//
	// [Ja] 試行が threshold より前に始まった started: Worker がタイムアウトを
	// 超えたため、回復の対象として返る必要がある。
	oldest := set.add(model.ExportStatusStarted, threshold.Add(-2*time.Hour))
	tied := []model.ExportID{
		set.add(model.ExportStatusStarted, threshold.Add(-time.Hour)),
		set.add(model.ExportStatusStarted, threshold.Add(-time.Hour)),
	}
	sortExportIDs(tied)

	// started_at exactly at the threshold is excluded by the strict < (boundary).
	//
	// [Ja] started_at がちょうど threshold の行は厳密な < により除外される (境界)。
	set.add(model.ExportStatusStarted, threshold)
	set.add(model.ExportStatusStarted, threshold.Add(time.Hour))
	// Non-started rows are out of scope regardless of age.
	//
	// [Ja] started でない行は経過時間に関わらず対象外。
	set.add(model.ExportStatusQueued, threshold.Add(-time.Hour))
	set.add(model.ExportStatusSucceeded, threshold.Add(-time.Hour))

	// Started on a profile whose deletion has started: generation stops at the
	// marker, so requeueing this row would only produce a job that returns
	// without touching it. Profile deletion is what removes the row.
	//
	// [Ja] 削除が始まったプロフィールの started: 生成はマーカーで止まるため、この行を
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
		t.Fatalf("ListStaleStarted() error = %v", err)
	}
	assertExportIDsInOrder(t, set.owned(got), oldest, tied[0], tied[1])
	assertNoNextCursor(t, next)

	// Asserted against the unfiltered result: set.owned would drop this row
	// whether the query returned it or not, so it cannot show the exclusion.
	//
	// [Ja] 絞り込み前の結果に対して検証する。set.owned はクエリが返したかどうかに
	// 関わらずこの行を落とすため、除外されたことを示せないため。
	if slices.Contains(exportIDs(got), deleting) {
		t.Errorf("削除開始済みプロフィールの started が返っている: %v", deleting)
	}

	// Page through the raw global result to assert the exact bound and that the
	// cursor follows started_at, not created_at.
	//
	// [Ja] グローバルな結果をページングし、正確な上限と、cursor が created_at では
	// なく started_at に従うことを検証する。
	firstPage, cursor, err := repo.ListStaleStarted(ctx, threshold, nil, 2)
	if err != nil {
		t.Fatalf("ListStaleStarted() first page error = %v", err)
	}
	assertExportIDsInOrder(t, firstPage, exportIDs(got[:2])...)
	assertNextCursor(t, cursor, *firstPage[len(firstPage)-1].StartedAt, firstPage[len(firstPage)-1].ID)

	secondPage, _, err := repo.ListStaleStarted(ctx, threshold, cursor, 2)
	if err != nil {
		t.Fatalf("ListStaleStarted() second page error = %v", err)
	}
	assertExportIDsInOrder(t, secondPage, exportIDs(got[2:min(4, len(got))])...)
}

func TestExportRepository_ListOldSucceededByProfileID(t *testing.T) {
	t.Parallel()

	t.Run("最新以外の succeeded を返し、最新を除外する", func(t *testing.T) {
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
			t.Fatalf("ListOldSucceededByProfileID() error = %v", err)
		}
		assertExportIDsInOrder(t, got, old1, old2)
		for _, e := range got {
			if e.ID == latest {
				t.Errorf("latest succeeded %v must not be selected for deletion", latest)
			}
		}
	})

	t.Run("同一 created_at では id で tie-break し、最新 (最大 id) を除外する", func(t *testing.T) {
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
			t.Fatalf("ListOldSucceededByProfileID() error = %v", err)
		}
		assertExportIDsInOrder(t, got, ids[:len(ids)-1]...)
		for _, e := range got {
			if e.ID == latest {
				t.Errorf("latest succeeded %v (max id) must not be selected for deletion", latest)
			}
		}
	})

	t.Run("非 succeeded と他プロフィールを候補にしない", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		old := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(2*time.Hour)) // latest
		// failed / queued on the same profile must not become cleanup candidates.
		//
		// [Ja] 同一プロフィールの failed / queued は cleanup 候補にしてはならない。
		buildExport(t, tx, profileID, actorID, model.ExportStatusFailed, base.Add(time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusQueued, base.Add(3*time.Hour))

		// A succeeded on another profile must not leak in.
		//
		// [Ja] 別プロフィールの succeeded が混入してはならない。
		otherOwner := testutil.NewProfileOwner(t, tx)
		otherProfile, otherActor := otherOwner.ProfileID, otherOwner.ActorID
		buildExport(t, tx, otherProfile, otherActor, model.ExportStatusSucceeded, base)
		buildExport(t, tx, otherProfile, otherActor, model.ExportStatusSucceeded, base.Add(time.Hour))

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID() error = %v", err)
		}
		assertExportIDsInOrder(t, got, old)
	})

	t.Run("succeeded が 1 件だけなら空を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID

		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, time.Now())

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, unboundedTestPageSize)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("returned %d exports, want 0", len(got))
		}
	})

	t.Run("page size を超える候補は切り捨て、古い順の先頭を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		// Three old succeeded plus the latest: cleanup that ran with page size 2
		// must take the two oldest, and the rest is picked up on the next run
		// once these are deleted.
		//
		// [Ja] 古い succeeded 3 件 + 最新 1 件。page size 2 で動いた cleanup は最も
		// 古い 2 件を取り、残りはそれらが削除された次回の実行で拾われる。
		old1 := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		old2 := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(2*time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(3*time.Hour))

		got, err := repo.ListOldSucceededByProfileID(ctx, profileID, 2)
		if err != nil {
			t.Fatalf("ListOldSucceededByProfileID() error = %v", err)
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

	// Two succeeded: has old rows to clean up, so this profile must be returned.
	//
	// [Ja] succeeded 2 件: 掃除すべき古い行があるため、このプロフィールは返る。
	ownerTwo := testutil.NewProfileOwner(t, tx)
	withTwo, actorTwo := ownerTwo.ProfileID, ownerTwo.ActorID
	buildExport(t, tx, withTwo, actorTwo, model.ExportStatusSucceeded, base)
	buildExport(t, tx, withTwo, actorTwo, model.ExportStatusSucceeded, base.Add(time.Hour))

	// A second eligible profile ensures page size 1 has a real next page.
	//
	// [Ja] 2 件目の対象プロフィールにより、page size 1 の次ページが実在することを
	// 保証する。
	ownerTwoSecond := testutil.NewProfileOwner(t, tx)
	withTwoSecond, actorTwoSecond := ownerTwoSecond.ProfileID, ownerTwoSecond.ActorID
	buildExport(t, tx, withTwoSecond, actorTwoSecond, model.ExportStatusSucceeded, base)
	buildExport(t, tx, withTwoSecond, actorTwoSecond, model.ExportStatusSucceeded, base.Add(time.Hour))

	// One succeeded: nothing old, so this profile must not be returned.
	//
	// [Ja] succeeded 1 件: 古い行が無いため、このプロフィールは返らない。
	ownerOne := testutil.NewProfileOwner(t, tx)
	withOne, actorOne := ownerOne.ProfileID, ownerOne.ActorID
	buildExport(t, tx, withOne, actorOne, model.ExportStatusSucceeded, base)

	// One succeeded plus a failed: still only one succeeded, so not returned.
	//
	// [Ja] succeeded 1 件 + failed 1 件: succeeded は依然 1 件のため返らない。
	ownerFailed := testutil.NewProfileOwner(t, tx)
	withFailed, actorFailed := ownerFailed.ProfileID, ownerFailed.ActorID
	buildExport(t, tx, withFailed, actorFailed, model.ExportStatusSucceeded, base)
	buildExport(t, tx, withFailed, actorFailed, model.ExportStatusFailed, base.Add(time.Hour))

	got, next, err := repo.ListProfileIDsWithOldSucceeded(ctx, nil, unboundedTestPageSize)
	if err != nil {
		t.Fatalf("ListProfileIDsWithOldSucceeded() error = %v", err)
	}
	if next != nil {
		t.Errorf("next cursor = %v, want nil", next)
	}
	gotSet := make(map[model.ProfileID]bool, len(got))
	for _, id := range got {
		gotSet[id] = true
	}
	if !gotSet[withTwo] {
		t.Errorf("profile with two succeeded %v missing from result", withTwo)
	}
	if !gotSet[withTwoSecond] {
		t.Errorf("second profile with two succeeded %v missing from result", withTwoSecond)
	}
	if gotSet[withOne] {
		t.Errorf("profile with one succeeded %v must not be returned", withOne)
	}
	if gotSet[withFailed] {
		t.Errorf("profile with one succeeded and one failed %v must not be returned", withFailed)
	}

	if !slices.IsSortedFunc(got, func(a, b model.ProfileID) int {
		ua, ub := uuid.UUID(a), uuid.UUID(b)
		return bytes.Compare(ua[:], ub[:])
	}) {
		t.Errorf("ListProfileIDsWithOldSucceeded() result is not sorted: %v", got)
	}

	// Page through the raw global result to prove that a page-size bound cannot
	// strand profiles after the first page.
	//
	// [Ja] グローバルな結果をページングし、ページサイズ上限によって 2 ページ目
	// 以降のプロフィールが取り残されないことを検証する。
	firstPage, cursor, err := repo.ListProfileIDsWithOldSucceeded(ctx, nil, 1)
	if err != nil {
		t.Fatalf("ListProfileIDsWithOldSucceeded() first page error = %v", err)
	}
	if len(firstPage) != 1 || firstPage[0] != got[0] {
		t.Fatalf("first page = %v, want [%v]", firstPage, got[0])
	}
	if cursor == nil || *cursor != firstPage[0] {
		t.Fatalf("next cursor = %v, want %v", cursor, firstPage[0])
	}

	secondPage, _, err := repo.ListProfileIDsWithOldSucceeded(ctx, cursor, 1)
	if err != nil {
		t.Fatalf("ListProfileIDsWithOldSucceeded() second page error = %v", err)
	}
	if len(secondPage) != 1 || secondPage[0] != got[1] {
		t.Fatalf("second page = %v, want [%v]", secondPage, got[1])
	}
}

func TestExportRepository_FindIDsRetainingObject(t *testing.T) {
	t.Parallel()

	t.Run("オブジェクトを保持している ID だけを返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		queued := newExportWithStatus(t, tx, model.ExportStatusQueued, time.Now())
		started := newExportWithStatus(t, tx, model.ExportStatusStarted, time.Now())
		succeeded := newExportWithStatus(t, tx, model.ExportStatusSucceeded, time.Now())
		// A failed export has released its object: the terminal transition is what
		// gives it up, so an object still stored under this ID is an orphan the
		// sweep has to collect rather than one this row keeps alive.
		//
		// [Ja] failed のエクスポートはオブジェクトを手放している。手放すのが終端遷移で
		// あるため、この ID でまだ保存されているオブジェクトは、この行が生かしている
		// ものではなく、掃除が回収すべき孤児である。
		failed := newExportWithStatus(t, tx, model.ExportStatusFailed, time.Now())
		missing := model.ExportID(uuid.New())

		got, err := repo.FindIDsRetainingObject(ctx, []model.ExportID{queued, started, succeeded, failed, missing})
		if err != nil {
			t.Fatalf("FindIDsRetainingObject() error = %v", err)
		}
		gotSet := make(map[model.ExportID]bool, len(got))
		for _, id := range got {
			gotSet[id] = true
		}
		if len(gotSet) != 3 || !gotSet[queued] || !gotSet[started] || !gotSet[succeeded] {
			t.Errorf("FindIDsRetainingObject() = %v, want {%v, %v, %v}", got, queued, started, succeeded)
		}
		if gotSet[failed] {
			t.Errorf("failed のエクスポート %v が保持側に含まれています", failed)
		}
		if gotSet[missing] {
			t.Errorf("存在しない ID %v が結果に含まれています", missing)
		}
	})

	t.Run("空入力は nil を返しクエリを発行しない", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		got, err := repo.FindIDsRetainingObject(ctx, nil)
		if err != nil {
			t.Fatalf("FindIDsRetainingObject() error = %v", err)
		}
		if got != nil {
			t.Errorf("FindIDsRetainingObject(nil) = %v, want nil", got)
		}
	})
}

// TestExportRepository_ListByProfileID pins what a profile removal is handed to
// delete. Every status has to be there: an attempt uploads before the
// transition that records it, so a queued, started or failed export can own an
// object too, and a listing that left those out would let a deleted profile
// leave archives nothing points at.
//
// [Ja] TestExportRepository_ListByProfileID は、プロフィールの削除処理が削除対象と
// して受け取るものを固定する。全 status が含まれる必要がある。試行はそれを記録する
// 遷移より先にアップロードするため、queued / started / failed のエクスポートも
// オブジェクトを持ちうる。それらを除いた一覧は、削除されたプロフィールに、どこからも
// 指されないアーカイブを残させることになる。
func TestExportRepository_ListByProfileID(t *testing.T) {
	t.Parallel()

	t.Run("status を問わず古い順に返し、他プロフィールを含めない", func(t *testing.T) {
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
			t.Fatalf("ListByProfileID() error = %v", err)
		}
		assertExportIDsInOrder(t, got, succeeded, failed, queued)
	})

	t.Run("page size を超える行は切り捨て、古い順の先頭を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
		owner := testutil.NewProfileOwner(t, tx)
		profileID, actorID := owner.ProfileID, owner.ActorID
		base := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)

		// The caller deletes the rows it processed, so the remaining ones appear
		// at the head of the next query and no cursor is needed.
		//
		// [Ja] 呼び出し側は処理した行を削除するため、残りは次のクエリの先頭に現れ、
		// cursor は不要になる。
		first := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base)
		second := buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(time.Hour))
		buildExport(t, tx, profileID, actorID, model.ExportStatusSucceeded, base.Add(2*time.Hour))

		got, err := repo.ListByProfileID(ctx, profileID, 2)
		if err != nil {
			t.Fatalf("ListByProfileID() error = %v", err)
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
			t.Fatalf("ListByProfileID() error = %v", err)
		}
		if len(got) != 0 {
			t.Errorf("returned %d exports, want 0", len(got))
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
				t.Fatalf("Delete() error = %v", err)
			}
			if !deleted {
				t.Fatal("Delete() = false, want true")
			}

			remaining, err := repo.FindLatestByProfileID(ctx, profileID)
			if err != nil {
				t.Fatalf("FindLatestByProfileID() error = %v", err)
			}
			if remaining != nil {
				t.Errorf("export still present after Delete: %v", remaining.ID)
			}
		})
	}

	t.Run("存在しない行の削除は false を返す", func(t *testing.T) {
		_, tx := testutil.SetupTx(t)
		ctx := context.Background()
		repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

		deleted, err := repo.Delete(ctx, model.ExportID(uuid.New()))
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if deleted {
			t.Error("Delete() on a missing row = true, want false")
		}
	})
}
