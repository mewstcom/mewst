package repository_test

import (
	"bytes"
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// mustLoadLocation resolves an IANA time zone for the export listings, which
// take a location rather than a raw name.
//
// [Ja] mustLoadLocation はエクスポート用の一覧が名前ではなく location を受け取る
// ため、IANA タイムゾーンを解決する。
func mustLoadLocation(t testing.TB, name string) *time.Location {
	t.Helper()

	location, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("タイムゾーンの解決に失敗: %v", err)
	}
	return location
}

// newExportPostFixture creates an export target and returns a builder factory
// that inserts posts for its profile, so each test only states the published
// time it cares about.
//
// [Ja] newExportPostFixture はエクスポート対象を作成し、そのプロフィールの投稿を
// 挿入するビルダーの生成関数を返す。各テストは関心のある公開日時だけを書けばよい。
func newExportPostFixture(t *testing.T, tx *sql.Tx) (model.ProfileID, model.ActorID, func(publishedAt time.Time) *testutil.PostBuilder) {
	t.Helper()

	owner := testutil.NewProfileOwner(t, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID
	oauthApplicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()

	return profileID, actorID, func(publishedAt time.Time) *testutil.PostBuilder {
		return testutil.NewPostBuilder(t, tx).
			WithProfileID(profileID).
			WithOauthApplicationID(oauthApplicationID).
			WithPublishedAt(publishedAt)
	}
}

// createExportSnapshot creates an export whose export_posts rows were
// materialized by ExportRepository.Create.
//
// [Ja] createExportSnapshot は export_posts を ExportRepository.Create が
// 固定化した export を作成する。
func createExportSnapshot(t testing.TB, tx *sql.Tx, profileID model.ProfileID, actorID model.ActorID) *model.Export {
	t.Helper()

	export, err := repository.NewExportRepository(testutil.QueriesWithTx(tx)).Create(
		context.Background(),
		repository.CreateExportInput{ProfileID: profileID, ActorID: actorID},
	)
	if err != nil {
		t.Fatalf("export snapshot の作成に失敗: %v", err)
	}
	return export
}

// failExportSnapshot walks the export to failed through started, so the
// partial unique index on active statuses lets the profile create another one.
// Reaching a terminal status also discards the export's snapshot, so a test
// that needs to read it has to do so before calling this.
//
// [Ja] failExportSnapshot は export を started 経由で failed へ進める。active な
// status に対する部分ユニークインデックスが、同じプロフィールで次の export を
// 作れるようにするため。終端状態への到達はその export の snapshot も破棄するため、
// snapshot を読むテストは本関数を呼ぶ前に読む必要がある。
func failExportSnapshot(t testing.TB, tx *sql.Tx, export *model.Export) {
	t.Helper()

	ctx := context.Background()
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))

	started, err := repo.MarkStarted(ctx, export.ID, export.UpdatedAt)
	if err != nil {
		t.Fatalf("started への遷移に失敗: %v", err)
	}
	if started == nil {
		// Fatal ends the test, but the helper takes testing.TB, so the
		// analyzer cannot tell that and the explicit return is what tells it.
		//
		// [Ja] Fatal はテストを終了させるが、本ヘルパーは testing.TB を受け取るため
		// 解析器がそれを判断できない。明示的な return がそれを伝える。
		t.Fatal("started への遷移でガードが一致しなかった")
		return
	}

	updated, err := repo.MarkFailed(ctx, started.ID, started.UpdatedAt)
	if err != nil {
		t.Fatalf("failed への遷移に失敗: %v", err)
	}
	if !updated {
		t.Fatal("failed への遷移でガードが一致しなかった")
	}
}

func TestExportPostRepository_ListMonthsByExportID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportPostRepository(testutil.QueriesWithTx(tx))

	t.Run("月の境界は対象タイムゾーンで決まる", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		location := mustLoadLocation(t, "Asia/Tokyo")
		// 2026-06-30T14:59:59Z is 23:59:59 on June 30 in JST, while
		// 2026-06-30T15:00:00Z is 00:00:00 on July 1.
		//
		// [Ja] 2026-06-30T14:59:59Z は JST では 6 月末日の 23:59:59、
		// 2026-06-30T15:00:00Z は JST では 7 月 1 日の 00:00:00。
		newPost(time.Date(2026, 6, 30, 14, 59, 59, 0, time.UTC)).Build()
		newPost(time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC)).Build()
		export := createExportSnapshot(t, tx, profileID, actorID)

		months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
			ExportID: export.ID,
			Location: location,
		})
		if err != nil {
			t.Fatalf("ListMonthsByExportID() error = %v", err)
		}

		want := []repository.PostMonth{
			{
				LocalMonthStart: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
				StartsAt:        time.Date(2026, 6, 30, 14, 59, 59, 0, time.UTC),
				EndsAt:          time.Date(2026, 6, 30, 14, 59, 59, 0, time.UTC).Add(time.Microsecond),
				PostCount:       1,
				Location:        location,
			},
			{
				LocalMonthStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				StartsAt:        time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC),
				EndsAt:          time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC).Add(time.Microsecond),
				PostCount:       1,
				Location:        location,
			},
		}
		assertPostMonths(t, months, want)
	})

	t.Run("夏時間をまたぐ月は境界ごとのオフセットで範囲が決まる", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		location := mustLoadLocation(t, "America/New_York")
		// US daylight saving time starts on March 8 in 2026. March starts in
		// EST (-05:00) and ends in EDT (-04:00).
		//
		// [Ja] 2026 年の米国夏時間は 3 月 8 日開始。3 月は EST (-05:00) で始まり、
		// EDT (-04:00) で終わるため、月の開始と終了でオフセットが異なる。
		newPost(time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC)).Build()
		newPost(time.Date(2026, 4, 1, 3, 59, 59, 0, time.UTC)).Build()
		newPost(time.Date(2026, 4, 1, 4, 0, 0, 0, time.UTC)).Build()
		export := createExportSnapshot(t, tx, profileID, actorID)

		months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
			ExportID: export.ID,
			Location: location,
		})
		if err != nil {
			t.Fatalf("ListMonthsByExportID() error = %v", err)
		}

		want := []repository.PostMonth{
			{
				LocalMonthStart: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				StartsAt:        time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC),
				EndsAt:          time.Date(2026, 4, 1, 3, 59, 59, 0, time.UTC).Add(time.Microsecond),
				PostCount:       2,
				Location:        location,
			},
			{
				LocalMonthStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
				StartsAt:        time.Date(2026, 4, 1, 4, 0, 0, 0, time.UTC),
				EndsAt:          time.Date(2026, 4, 1, 4, 0, 0, 0, time.UTC).Add(time.Microsecond),
				PostCount:       1,
				Location:        location,
			},
		}
		assertPostMonths(t, months, want)
	})

	t.Run("月初の DST フォールドでも集計した投稿をすべて返す", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		location := mustLoadLocation(t, "America/Havana")
		firstPublishedAt := time.Date(2020, 11, 1, 4, 30, 0, 0, time.UTC)
		secondPublishedAt := time.Date(2020, 11, 1, 5, 30, 0, 0, time.UTC)
		wantIDs := []model.PostID{
			newPost(firstPublishedAt).Build(),
			newPost(secondPublishedAt).Build(),
		}
		export := createExportSnapshot(t, tx, profileID, actorID)

		months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
			ExportID: export.ID,
			Location: location,
		})
		if err != nil {
			t.Fatalf("ListMonthsByExportID() error = %v", err)
		}
		wantMonths := []repository.PostMonth{
			{
				LocalMonthStart: time.Date(2020, 11, 1, 0, 0, 0, 0, time.UTC),
				StartsAt:        firstPublishedAt,
				EndsAt:          secondPublishedAt.Add(time.Microsecond),
				PostCount:       2,
				Location:        location,
			},
		}
		assertPostMonths(t, months, wantMonths)

		posts, next, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
			ExportID: export.ID,
			Month:    months[0],
			PageSize: 10,
		})
		if err != nil {
			t.Fatalf("ListByExportIDInRange() error = %v", err)
		}
		if next != nil {
			t.Errorf("next = %+v, want nil", next)
		}
		assertPostIDs(t, posts, wantIDs)
	})

	t.Run("discard 済みの投稿は件数にも月にも含まれない", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		newPost(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)).Build()
		newPost(time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)).
			WithDiscardedAt(time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)).
			Build()
		newPost(time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)).
			WithDiscardedAt(time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)).
			Build()
		export := createExportSnapshot(t, tx, profileID, actorID)

		months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
			ExportID: export.ID,
			Location: time.UTC,
		})
		if err != nil {
			t.Fatalf("ListMonthsByExportID() error = %v", err)
		}

		want := []repository.PostMonth{
			{
				LocalMonthStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				StartsAt:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				EndsAt:          time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Add(time.Microsecond),
				PostCount:       1,
				Location:        time.UTC,
			},
		}
		assertPostMonths(t, months, want)
	})

	t.Run("元投稿の物理削除と後発投稿がsnapshotを変えない", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		keptID := newPost(time.Date(2026, 7, 1, 1, 0, 0, 0, time.UTC)).
			WithContent("申請時点の本文").
			Build()
		otherKeptID := newPost(time.Date(2026, 7, 1, 2, 0, 0, 0, time.UTC)).
			Build()
		newPost(time.Date(2026, 7, 1, 1, 30, 0, 0, time.UTC)).
			WithDiscardedAt(time.Date(2026, 7, 1, 3, 0, 0, 0, time.UTC)).
			Build()
		export := createExportSnapshot(t, tx, profileID, actorID)

		// Rails can hard-delete a post after discard. A post created after the
		// export request is also inside the month scan range. Neither change may
		// alter the materialized request-time set.
		//
		// [Ja] Rails は discard 後に投稿を物理削除し得る。export 申請後に作る投稿も
		// 月の走査範囲内に置く。どちらの変更も固定済みの申請時点集合を変えてはならない。
		if _, err := tx.Exec("DELETE FROM posts WHERE id = $1", uuid.UUID(keptID)); err != nil {
			t.Fatalf("元投稿の物理削除に失敗: %v", err)
		}
		newPost(time.Date(2026, 7, 1, 1, 45, 0, 0, time.UTC)).
			Build()

		months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
			ExportID: export.ID,
			Location: time.UTC,
		})
		if err != nil {
			t.Fatalf("ListMonthsByExportID() error = %v", err)
		}
		if len(months) != 1 {
			t.Fatalf("len(months) = %d, want 1", len(months))
		}
		if months[0].PostCount != 2 {
			t.Fatalf("months[0].PostCount = %d, want 2", months[0].PostCount)
		}

		posts, _, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
			ExportID: export.ID,
			Month:    months[0],
			PageSize: 10,
		})
		if err != nil {
			t.Fatalf("ListByExportIDInRange() error = %v", err)
		}
		assertPostIDs(t, posts, []model.PostID{keptID, otherKeptID})
		if posts[0].Content != "申請時点の本文" {
			t.Errorf("posts[0].Content = %q, want %q", posts[0].Content, "申請時点の本文")
		}
	})

	t.Run("同じプロフィールの export はそれぞれ申請時点の snapshot を返す", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)

		assertSnapshot := func(name string, exportID model.ExportID, wantIDs []model.PostID) {
			t.Helper()

			months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
				ExportID: exportID,
				Location: time.UTC,
			})
			if err != nil {
				t.Fatalf("%s: ListMonthsByExportID() error = %v", name, err)
			}
			if len(months) != 1 {
				t.Fatalf("%s: len(months) = %d, want 1", name, len(months))
			}
			if months[0].PostCount != int64(len(wantIDs)) {
				t.Errorf("%s: months[0].PostCount = %d, want %d", name, months[0].PostCount, len(wantIDs))
			}

			posts, _, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
				ExportID: exportID,
				Month:    months[0],
				PageSize: 10,
			})
			if err != nil {
				t.Fatalf("%s: ListByExportIDInRange() error = %v", name, err)
			}
			assertPostIDs(t, posts, wantIDs)
		}

		firstID := newPost(time.Date(2026, 7, 1, 1, 0, 0, 0, time.UTC)).Build()
		first := createExportSnapshot(t, tx, profileID, actorID)

		// The post added after the first request must stay out of the first
		// snapshot and appear in the second one. The first snapshot is read
		// while its export is still active, because reaching a terminal status
		// discards it.
		//
		// [Ja] 1 回目の申請より後に追加した投稿は、1 つ目の snapshot には入らず
		// 2 つ目に現れる必要がある。1 つ目の snapshot は export が active なうちに
		// 読む。終端状態への到達で破棄されるため。
		secondID := newPost(time.Date(2026, 7, 1, 2, 0, 0, 0, time.UTC)).Build()
		assertSnapshot("1 つ目の export", first.ID, []model.PostID{firstID})

		// A profile can only hold one active export, so the first one has to
		// reach a terminal status before the second request.
		//
		// [Ja] プロフィールが同時に持てる active な export は 1 件なので、2 回目の
		// 申請前に 1 件目を終端状態にする。
		failExportSnapshot(t, tx, first)
		second := createExportSnapshot(t, tx, profileID, actorID)
		assertSnapshot("2 つ目の export", second.ID, []model.PostID{firstID, secondID})
	})

	t.Run("投稿が無いプロフィールは空を返す", func(t *testing.T) {
		profileID, actorID, _ := newExportPostFixture(t, tx)
		export := createExportSnapshot(t, tx, profileID, actorID)

		months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
			ExportID: export.ID,
			Location: time.UTC,
		})
		if err != nil {
			t.Fatalf("ListMonthsByExportID() error = %v", err)
		}
		if len(months) != 0 {
			t.Errorf("len(months) = %d, want 0", len(months))
		}
	})
}

// assertPostMonths compares the listing against the expected months in order.
// Locations are compared by name because the listing copies the caller's
// pointer and a test states the zone it passed in.
//
// [Ja] assertPostMonths は一覧の結果を期待する月と順序込みで比較する。一覧は
// 呼び出し側の location ポインタをそのまま複製し、テストは渡したゾーンを書くため、
// location は名前で比較する。
func assertPostMonths(t *testing.T, got, want []repository.PostMonth) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("len(months) = %d, want %d (months = %+v)", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if !g.LocalMonthStart.Equal(w.LocalMonthStart) {
			t.Errorf("months[%d].LocalMonthStart = %v, want %v", i, g.LocalMonthStart, w.LocalMonthStart)
		}
		if !g.StartsAt.Equal(w.StartsAt) {
			t.Errorf("months[%d].StartsAt = %v, want %v", i, g.StartsAt, w.StartsAt)
		}
		if !g.EndsAt.Equal(w.EndsAt) {
			t.Errorf("months[%d].EndsAt = %v, want %v", i, g.EndsAt, w.EndsAt)
		}
		if g.PostCount != w.PostCount {
			t.Errorf("months[%d].PostCount = %d, want %d", i, g.PostCount, w.PostCount)
		}
		if g.Location.String() != w.Location.String() {
			t.Errorf("months[%d].Location = %v, want %v", i, g.Location, w.Location)
		}
	}
}

func TestExportPostRepository_ListByExportIDInRange(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	repo := repository.NewExportPostRepository(testutil.QueriesWithTx(tx))

	startsAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endsAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	// The month is stated directly instead of taken from ListMonthsByExportID
	// so each case can place posts on either side of a known boundary. The
	// listings agree on the same month in the ListMonthsByExportID tests.
	//
	// [Ja] 各ケースが既知の境界の内外へ投稿を置けるよう、月は
	// ListMonthsByExportID から取らずに直接組み立てる。両一覧が同じ月で一致する
	// ことは ListMonthsByExportID のテストで確認している。
	month := repository.PostMonth{
		LocalMonthStart: startsAt,
		StartsAt:        startsAt,
		EndsAt:          endsAt,
		Location:        time.UTC,
	}

	t.Run("範囲は半開区間で discard 済みと他プロフィールを除外する", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		wantIDs := []model.PostID{
			newPost(startsAt).Build(),
			newPost(endsAt.Add(-time.Second)).Build(),
		}
		newPost(startsAt.Add(-time.Second)).Build()
		newPost(endsAt).Build()
		newPost(startsAt.Add(time.Hour)).
			WithDiscardedAt(time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)).
			Build()

		_, _, newOtherPost := newExportPostFixture(t, tx)
		newOtherPost(startsAt.Add(2 * time.Hour)).Build()
		export := createExportSnapshot(t, tx, profileID, actorID)

		posts, next, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
			ExportID: export.ID,
			Month:    month,
			PageSize: 10,
		})
		if err != nil {
			t.Fatalf("ListByExportIDInRange() error = %v", err)
		}
		if next != nil {
			t.Errorf("next = %+v, want nil", next)
		}
		assertPostIDs(t, posts, wantIDs)
	})

	t.Run("同一 published_at は id で tie-break する", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		publishedAt := startsAt.Add(3 * time.Hour)
		for range 5 {
			newPost(publishedAt).Build()
		}
		export := createExportSnapshot(t, tx, profileID, actorID)

		posts, _, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
			ExportID: export.ID,
			Month:    month,
			PageSize: 10,
		})
		if err != nil {
			t.Fatalf("ListByExportIDInRange() error = %v", err)
		}
		if len(posts) != 5 {
			t.Fatalf("len(posts) = %d, want 5", len(posts))
		}
		for i := 1; i < len(posts); i++ {
			previous, current := uuid.UUID(posts[i-1].ID), uuid.UUID(posts[i].ID)
			if bytes.Compare(previous[:], current[:]) >= 0 {
				t.Errorf("posts[%d].ID = %v は posts[%d].ID = %v より後に並ぶべき", i, current, i-1, previous)
			}
		}
	})

	t.Run("cursor でページを辿ると全件をちょうど 1 回ずつ返す", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		// Share one published_at among 41 posts so many page boundaries cross
		// an ID tie-break, exposing omissions or duplicates over dozens of pages.
		//
		// [Ja] 41 件ごとに同じ published_at を共有させ、多数のページ境界が ID の
		// tie-break をまたぐ場合も取りこぼしや重複が起きないことを確かめる。
		const (
			postCount         = 1003
			postsPerPublishAt = 41
			pageSize          = 37
		)
		wantIDs := make([]model.PostID, 0, postCount)
		for i := range postCount {
			wantIDs = append(wantIDs, newPost(startsAt.Add(time.Duration(i/postsPerPublishAt)*time.Hour)).Build())
		}
		export := createExportSnapshot(t, tx, profileID, actorID)
		// The expected order is published_at first and id within each tie.
		//
		// [Ja] 期待する並びは published_at 順、同値の中では id 順。
		for start := 0; start < len(wantIDs); start += postsPerPublishAt {
			sortPostIDs(wantIDs[start:min(start+postsPerPublishAt, len(wantIDs))])
		}

		var (
			gotPosts  []*repository.ExportPost
			cursor    *repository.PostCursor
			pageCount int
		)
		for page := 0; page < postCount; page++ {
			pageCount++
			posts, next, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
				ExportID: export.ID,
				Month:    month,
				Cursor:   cursor,
				PageSize: pageSize,
			})
			if err != nil {
				t.Fatalf("ListByExportIDInRange() error = %v", err)
			}
			gotPosts = append(gotPosts, posts...)
			cursor = next
			if cursor == nil {
				break
			}
		}
		if cursor != nil {
			t.Fatalf("cursor = %+v, want nil (ページ走査が終端に達していない)", cursor)
		}
		wantPageCount := (postCount + pageSize - 1) / pageSize
		if pageCount != wantPageCount {
			t.Errorf("page count = %d, want %d", pageCount, wantPageCount)
		}
		assertPostIDs(t, gotPosts, wantIDs)
	})

	t.Run("境界に別のタイムゾーンの時刻を渡しても同じ行を返す", func(t *testing.T) {
		profileID, actorID, newPost := newExportPostFixture(t, tx)
		wantIDs := []model.PostID{newPost(startsAt).Build()}
		newPost(endsAt).Build()
		export := createExportSnapshot(t, tx, profileID, actorID)

		tokyo := mustLoadLocation(t, "Asia/Tokyo")
		posts, _, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
			ExportID: export.ID,
			Month: repository.PostMonth{
				LocalMonthStart: month.LocalMonthStart,
				StartsAt:        month.StartsAt.In(tokyo),
				EndsAt:          month.EndsAt.In(tokyo),
				Location:        month.Location,
			},
			PageSize: 10,
		})
		if err != nil {
			t.Fatalf("ListByExportIDInRange() error = %v", err)
		}
		assertPostIDs(t, posts, wantIDs)
	})
}

// sortPostIDs sorts post IDs the way PostgreSQL orders the uuid column, so a
// test can state the expected order of posts sharing one published_at.
//
// [Ja] sortPostIDs は PostgreSQL の uuid カラムと同じ順序で投稿 ID を並べ替える。
// published_at が同値の投稿の期待する並び順をテストが書けるようにするため。
func sortPostIDs(ids []model.PostID) {
	slices.SortFunc(ids, func(a, b model.PostID) int {
		left, right := uuid.UUID(a), uuid.UUID(b)
		return bytes.Compare(left[:], right[:])
	})
}

// assertPostIDs compares the listed posts against the expected IDs in order.
//
// [Ja] assertPostIDs は取得した投稿を期待する ID と順序込みで比較する。
func assertPostIDs(t *testing.T, got []*repository.ExportPost, want []model.PostID) {
	t.Helper()

	gotIDs := make([]model.PostID, len(got))
	for i, post := range got {
		gotIDs[i] = post.ID
	}
	if !slices.Equal(gotIDs, want) {
		t.Errorf("post IDs = %v, want %v", gotIDs, want)
	}
}
