package seed

import (
	"context"
	"database/sql"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// followPair is one edge of the follow graph, by the roles at either end.
//
// [Ja] followPair は、フォローの構成のうち 1 つの辺を、その両端の役割で表したもの。
type followPair struct {
	source seedRole
	target seedRole
}

// storedFollow is one follow read back out of the database.
//
// [Ja] storedFollow は、データベースから読み戻したフォロー 1 件。
type storedFollow struct {
	followedAt time.Time
	createdAt  time.Time
}

// timelineSourcePost is one of a profile's posts as the test reads them back,
// with whether a screen could still reach it.
//
// [Ja] timelineSourcePost は、テストが読み戻したプロフィールのポスト 1 件と、
// それが画面から辿り着ける状態にあるかどうか。
type timelineSourcePost struct {
	id          model.PostID
	publishedAt time.Time
	discarded   bool
}

// TestFollowEdges verifies that the graph is one the application could have
// been left in, and that it is not symmetric.
//
// [Ja] TestFollowEdges は、フォローの構成がアプリケーションによって残されえた
// ものであること、そして対称ではないことを検証する。
func TestFollowEdges(t *testing.T) {
	t.Parallel()

	seen := make(map[followPair]bool, len(followEdges))
	for _, edge := range followEdges {
		if edge.source == edge.target {
			t.Errorf("役割 %s が自分自身をフォローしている", edge.source)
		}

		if !slices.Contains(allSeedRoles, edge.source) || !slices.Contains(allSeedRoles, edge.target) {
			t.Errorf("フォロー %s -> %s に名簿の持たない役割が含まれる", edge.source, edge.target)
		}

		// The column pair carries a unique index, so a repeated edge is a row
		// the second write of it cannot make.
		//
		// [Ja] このカラムの組には一意インデックスがあるため、重複する辺は、
		// その 2 度目の書き込みが作れない行になる。
		pair := followPair{source: edge.source, target: edge.target}
		if seen[pair] {
			t.Errorf("フォロー %s -> %s が重複している", edge.source, edge.target)
		}
		seen[pair] = true

		if edge.monthsAgo <= 0 {
			t.Errorf("フォロー %s -> %s の monthsAgo = %d, want 1 以上", edge.source, edge.target, edge.monthsAgo)
		}
	}

	// A profile shows how many it follows and how many follow it. Where every
	// follow is returned, those two numbers are the same for everyone, and a
	// screen that read one of them for both would look right.
	//
	// [Ja] プロフィールは、フォローしている数とフォローされている数を示す。
	// すべてのフォローが返される構成では、この 2 つの数は誰にとっても同じであり、
	// 一方をもう一方にも読んでしまう画面が正しく見えることになる。
	var mutual, oneWay int
	for pair := range seen {
		if seen[followPair{source: pair.target, target: pair.source}] {
			mutual++
			continue
		}
		oneWay++
	}

	if mutual == 0 {
		t.Error("相互フォローの組が無い")
	}
	if oneWay == 0 {
		t.Error("片方向のフォローが無い")
	}

	// The account the suggestions are offered to has to be one with no
	// follows: the screen that offers them is what an account sees before it
	// has any.
	//
	// [Ja] おすすめが提示されるアカウントは、フォローを持たないものである必要が
	// ある。それを提示する画面は、アカウントがフォローを 1 つも持たないうちに見る
	// ものであるため。
	for _, edge := range followEdges {
		if edge.source == suggestedFollowSource {
			t.Errorf("おすすめの提示先である役割 %s が %s をフォローしている", edge.source, edge.target)
		}
	}
}

// TestSuggestedFollowTargets verifies that every suggestion is one the screen
// would display.
//
// [Ja] TestSuggestedFollowTargets は、おすすめのいずれもが画面に表示されるもので
// あることを検証する。
func TestSuggestedFollowTargets(t *testing.T) {
	t.Parallel()

	seen := make(map[seedRole]bool, len(suggestedFollowTargets))
	for _, role := range suggestedFollowTargets {
		if seen[role] {
			t.Errorf("おすすめの役割 %s が重複している", role)
		}
		seen[role] = true

		if role == suggestedFollowSource {
			t.Errorf("役割 %s が自分自身をおすすめされている", role)
		}

		// The screen filters the suggestions down to the profiles that are
		// still there, so a deleted one would be a row nothing displays.
		//
		// [Ja] 画面はおすすめを、まだ残っているプロフィールへ絞り込む。削除済みの
		// ものは、何にも表示されない行になる。
		if role == roleDiscarded {
			t.Errorf("削除済みプロフィールの役割 %s がおすすめに含まれている", role)
		}

		if !slices.Contains(allSeedRoles, role) {
			t.Errorf("おすすめに名簿の持たない役割 %s が含まれる", role)
		}
	}
}

// TestCreateFollows verifies that the graph reaches the database as it is
// written, that each profile's home timeline holds what its follows gave it,
// and that the account with no follows is offered the suggestions instead.
//
// [Ja] TestCreateFollows は、フォローの構成が書かれたとおりにデータベースへ届く
// こと、各プロフィールのホームタイムラインがそのフォローの与えたものを持つこと、
// そしてフォローを持たないアカウントには代わりにおすすめが提示されることを検証する。
func TestCreateFollows(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// The instant is truncated to what the column can hold, so that a
	// followed_at read back out of the database can be compared with the one
	// that was written rather than with a value rounded away from it.
	//
	// [Ja] 時点はカラムが保持できる精度へ切り詰める。データベースから読み戻した
	// followed_at を、そこから丸められた値ではなく、書き込んだ値と比較できるように
	// するため。
	now := time.Now().Truncate(time.Microsecond)

	accounts, err := createAccounts(ctx, tx, newTestRoster(t), now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	applicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()
	if err := createPosts(ctx, tx, testAmounts, applicationID, accounts, now); err != nil {
		t.Fatalf("ポストの作成に失敗: %v", err)
	}

	if err := createFollows(ctx, tx, accounts, now); err != nil {
		t.Fatalf("フォローの作成に失敗: %v", err)
	}

	assertFollowRows(t, ctx, tx, accounts, now)
	assertHomeTimelines(t, ctx, tx, accounts, now)
	assertSuggestedFollows(t, ctx, tx, accounts, now)
}

// assertFollowRows compares the follows in the database with the graph.
//
// [Ja] assertFollowRows は、データベースのフォローをフォローの構成と突き合わせる。
func assertFollowRows(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount, now time.Time) {
	t.Helper()

	roles := rolesByProfileID(t, accounts)

	rows, err := tx.QueryContext(ctx, `
		SELECT source_profile_id, target_profile_id, followed_at, created_at
		FROM follows
	`)
	if err != nil {
		t.Fatalf("フォローの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	got := make(map[followPair]storedFollow)
	for rows.Next() {
		var sourceID, targetID uuid.UUID
		var follow storedFollow

		if err := rows.Scan(&sourceID, &targetID, &follow.followedAt, &follow.createdAt); err != nil {
			t.Fatalf("フォローの読み取りに失敗: %v", err)
		}

		got[followPair{source: roles[model.ProfileID(sourceID)], target: roles[model.ProfileID(targetID)]}] = follow
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("フォローの取得に失敗: %v", err)
	}

	if len(got) != len(followEdges) {
		t.Fatalf("フォローの件数 = %d, want %d", len(got), len(followEdges))
	}

	for _, edge := range followEdges {
		pair := followPair{source: edge.source, target: edge.target}

		follow, ok := got[pair]
		if !ok {
			t.Errorf("フォロー %s -> %s が作成されていない", edge.source, edge.target)
			continue
		}

		want := storedInstant(now.AddDate(0, -edge.monthsAgo, 0))
		if !follow.followedAt.Equal(want) {
			t.Errorf("フォロー %s -> %s の followed_at = %v, want %v", edge.source, edge.target, follow.followedAt, want)
		}

		// A follow is created at the moment it is made. The screens that order
		// follows read created_at, so a row created at a different instant
		// than it was made at would be listed in an order followed_at
		// contradicts.
		//
		// [Ja] フォローは、それが行われた瞬間に作成される。フォローを並べる画面は
		// created_at を読むため、行われた時点と違う時点で作成された行は、
		// followed_at が否定する順序で並べられることになる。
		if !follow.createdAt.Equal(want) {
			t.Errorf("フォロー %s -> %s の created_at = %v, want %v", edge.source, edge.target, follow.createdAt, want)
		}
	}
}

// assertHomeTimelines compares each profile's home timeline with what its own
// posts and its follows gave it.
//
// The expectation is worked out in Go from the posts that were written, rather
// than by asking the database the question the generator asked it. A timeline
// checked against the query that filled it would agree with itself whatever
// that query says.
//
// [Ja] assertHomeTimelines は、各プロフィールのホームタイムラインを、自身のポストと
// そのフォローが与えたものと突き合わせる。
//
// 期待する内容は、生成器がデータベースへ投げた問いを投げ直すのではなく、書き込まれた
// ポストから Go の側で組み立てる。それを埋めたクエリと突き合わせたタイムラインは、
// そのクエリが何を言おうと自分自身と一致してしまう。
func assertHomeTimelines(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount, now time.Time) {
	t.Helper()

	// The cap is only exercised where the target had more posts than it than
	// the follow could take. It is counted while the expectations are built,
	// and checked afterwards, so that a smaller testAmounts fails here rather
	// than quietly stopping the test from reaching the branch.
	//
	// [Ja] 取り込みの上限が効くのは、相手がそれを超える件数のポストを既に持って
	// いた場合だけである。期待値を組み立てながら数え、あとで確認する。testAmounts を
	// 小さくしたときに、テストがその分岐へ届かなくなるのではなく、ここで失敗する
	// ようにするため。
	capped := 0

	// An account that follows nobody still has its own posts on its home
	// screen. It is counted the same way, so that a graph which came to give
	// every account with posts a follow fails here rather than quietly leaving
	// the own-posts side of a timeline checked only where a follow also filled
	// it.
	//
	// [Ja] 誰もフォローしていないアカウントも、自身のポストはホーム画面に持つ。
	// これも同じように数える。ポストを持つすべてのアカウントがフォローを持つように
	// なった構成が、タイムラインの自身のポストの側を、フォローもそれを埋めた場所で
	// しか検査しない状態を静かに残すのではなく、ここで失敗するようにするため。
	ownOnly := 0

	for _, account := range accounts {
		want := wantOwnTimelinePosts(t, ctx, tx, account)
		follows := false

		for _, edge := range followEdges {
			if edge.source != account.roster.role {
				continue
			}
			follows = true

			target, err := accountForRole(accounts, edge.target)
			if err != nil {
				t.Fatalf("役割 %s のアカウントの取得に失敗: %v", edge.target, err)
			}

			followedAt := storedInstant(now.AddDate(0, -edge.monthsAgo, 0))
			delivered, backfilled, available := wantTimelinePosts(t, ctx, tx, target, followedAt)

			if available > backfilledTimelinePosts {
				capped++
			}

			for id, publishedAt := range delivered {
				want[id] = publishedAt
			}
			for id, publishedAt := range backfilled {
				want[id] = publishedAt
			}
		}

		if !follows && len(want) > 0 {
			ownOnly++
		}

		got := readHomeTimeline(t, ctx, tx, account.profile.ID)

		if len(got) != len(want) {
			t.Errorf("役割 %s のホームタイムラインの件数 = %d, want %d", account.roster.role, len(got), len(want))
		}

		for id, publishedAt := range want {
			stored, ok := got[id]
			if !ok {
				t.Errorf("役割 %s のホームタイムラインにポスト %s が無い", account.roster.role, uuid.UUID(id))
				continue
			}

			// The timeline carries the publication time so that it can be
			// ordered without reaching for the post. A row holding another
			// instant would put the post somewhere the profile does not.
			//
			// [Ja] タイムラインは、ポストを参照せずに並べ替えられるよう公開日時を
			// 持つ。別の時点を持つ行は、そのポストをプロフィールが置いていない位置へ
			// 置くことになる。
			if !stored.Equal(publishedAt) {
				t.Errorf("役割 %s のホームタイムラインのポスト %s の published_at = %v, want %v",
					account.roster.role, uuid.UUID(id), stored, publishedAt)
			}
		}

		for id := range got {
			if _, ok := want[id]; !ok {
				t.Errorf("役割 %s のホームタイムラインに余分なポスト %s がある", account.roster.role, uuid.UUID(id))
			}
		}
	}

	if capped == 0 {
		t.Errorf("既にあったポストの取り込みが上限 (%d 件) に達するフォローが無い", backfilledTimelinePosts)
	}
	if ownOnly == 0 {
		t.Error("フォローを持たないまま自身のポストがホームタイムラインに並ぶアカウントが無い")
	}
}

// wantOwnTimelinePosts returns the posts an account should have put on its own
// home timeline: the ones it holds that a screen can still reach.
//
// [Ja] wantOwnTimelinePosts は、あるアカウントが自身のホームタイムラインへ置いたはず
// のものを返す。そのアカウントが持つポストのうち、画面から辿り着けるものになる。
func wantOwnTimelinePosts(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	account seedAccount,
) map[model.PostID]time.Time {
	t.Helper()

	own := make(map[model.PostID]time.Time)

	// Nobody signs in as a deleted profile, so its own home screen is one no
	// run opens and no row is written for it.
	//
	// [Ja] 削除済みプロフィールでサインインする人はいない。その自身のホーム画面は
	// どの実行も開かないものであり、そのための行も書かれない。
	if account.profile.DiscardedAt != nil {
		return own
	}

	for _, post := range readTimelineSourcePosts(t, ctx, tx, account.profile.ID) {
		// A deleted post takes its timeline rows with it, the writer's own
		// included.
		//
		// [Ja] 削除されたポストはタイムラインの行も一緒に取り除く。書いた本人の分も
		// そこに含まれる。
		if post.discarded {
			continue
		}

		own[post.id] = post.publishedAt
	}

	return own
}

// wantTimelinePosts returns what a follow made at followedAt should have put
// into the follower's timeline: everything the target published since, and the
// newest backfilledTimelinePosts of what it had published before. It also
// returns how many posts the second of those was chosen from.
//
// [Ja] wantTimelinePosts は、followedAt に行われたフォローがフォローした側の
// タイムラインへ入れたはずのものを返す。相手がそれ以降に公開したもののすべてと、
// それより前に公開していたもののうち新しい backfilledTimelinePosts 件になる。
// あわせて、後者が何件の中から選ばれたのかを返す。
func wantTimelinePosts(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	target seedAccount,
	followedAt time.Time,
) (delivered, backfilled map[model.PostID]time.Time, available int) {
	t.Helper()

	delivered = make(map[model.PostID]time.Time)
	backfilled = make(map[model.PostID]time.Time)

	// The posts of a deleted profile are not shown, so a follow of one gives
	// its follower nothing. No edge points at such a profile today, and this
	// is what says so rather than leaving the reader to work it out from the
	// counts.
	//
	// [Ja] 削除済みプロフィールのポストは表示されないため、そのプロフィールへの
	// フォローはフォローした側へ何も与えない。今日そうしたプロフィールを指す辺は
	// 無く、それを件数から読み取らせるのではなく、ここで述べる。
	if target.profile.DiscardedAt != nil {
		return delivered, backfilled, 0
	}

	var older []timelineSourcePost
	for _, post := range readTimelineSourcePosts(t, ctx, tx, target.profile.ID) {
		// A deleted post takes its timeline rows with it, so it is not one a
		// follow can hand over.
		//
		// [Ja] 削除されたポストはタイムラインの行も一緒に取り除くため、フォローが
		// 渡せるものではない。
		if post.discarded {
			continue
		}

		if post.publishedAt.Before(followedAt) {
			older = append(older, post)
			continue
		}

		delivered[post.id] = post.publishedAt
	}

	// The posts come back oldest first, so the newest of the older ones are
	// the tail of that list.
	//
	// [Ja] ポストは古いものから返るため、それより前のもののうち新しいものは、その
	// 一覧の末尾になる。
	newest := older
	if len(newest) > backfilledTimelinePosts {
		newest = newest[len(newest)-backfilledTimelinePosts:]
	}

	for _, post := range newest {
		backfilled[post.id] = post.publishedAt
	}

	return delivered, backfilled, len(older)
}

// readTimelineSourcePosts returns the profile's posts, oldest first.
//
// [Ja] readTimelineSourcePosts は、プロフィールのポストを古いものから順に返す。
func readTimelineSourcePosts(t *testing.T, ctx context.Context, tx *sql.Tx, profileID model.ProfileID) []timelineSourcePost {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT id, published_at, discarded_at IS NOT NULL
		FROM posts
		WHERE profile_id = $1
		ORDER BY published_at, id
	`, uuid.UUID(profileID))
	if err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var posts []timelineSourcePost
	for rows.Next() {
		var id uuid.UUID
		var post timelineSourcePost

		if err := rows.Scan(&id, &post.publishedAt, &post.discarded); err != nil {
			t.Fatalf("ポストの読み取りに失敗: %v", err)
		}

		post.id = model.PostID(id)
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}

	return posts
}

// readHomeTimeline returns the profile's home timeline, by post.
//
// [Ja] readHomeTimeline は、プロフィールのホームタイムラインをポストごとに返す。
func readHomeTimeline(t *testing.T, ctx context.Context, tx *sql.Tx, profileID model.ProfileID) map[model.PostID]time.Time {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT post_id, published_at
		FROM home_timeline_posts
		WHERE profile_id = $1
	`, uuid.UUID(profileID))
	if err != nil {
		t.Fatalf("ホームタイムラインの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	timeline := make(map[model.PostID]time.Time)
	for rows.Next() {
		var postID uuid.UUID
		var publishedAt time.Time

		if err := rows.Scan(&postID, &publishedAt); err != nil {
			t.Fatalf("ホームタイムラインの読み取りに失敗: %v", err)
		}

		timeline[model.PostID(postID)] = publishedAt
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ホームタイムラインの取得に失敗: %v", err)
	}

	return timeline
}

// assertSuggestedFollows verifies that the suggestions are offered to the
// account with no follows, in the order the screen lists them.
//
// [Ja] assertSuggestedFollows は、おすすめがフォローを持たないアカウントへ、画面が
// 並べる順序で提示されることを検証する。
func assertSuggestedFollows(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount, now time.Time) {
	t.Helper()

	roles := rolesByProfileID(t, accounts)

	source, err := accountForRole(accounts, suggestedFollowSource)
	if err != nil {
		t.Fatalf("役割 %s のアカウントの取得に失敗: %v", suggestedFollowSource, err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM suggested_follows`).Scan(&count); err != nil {
		t.Fatalf("おすすめフォローの件数の取得に失敗: %v", err)
	}
	if count != len(suggestedFollowTargets) {
		t.Errorf("おすすめフォローの件数 = %d, want %d", count, len(suggestedFollowTargets))
	}

	// Match Search::ShowController's profile association, including its
	// filters and sort key, so the assertion checks the visible order.
	//
	// [Ja] Search::ShowController のプロフィール関連と同じ絞り込み・ソートキーを
	// 使い、画面に表示される順序を検証する。
	rows, err := tx.QueryContext(ctx, `
		SELECT profiles.id, profiles.created_at, profiles.joined_at, suggested_follows.created_at
		FROM profiles
		JOIN suggested_follows ON profiles.id = suggested_follows.target_profile_id
		WHERE suggested_follows.source_profile_id = $1
		  AND profiles.discarded_at IS NULL
		  AND suggested_follows.checked_at IS NULL
		ORDER BY profiles.created_at DESC
		LIMIT 30
	`, uuid.UUID(source.profile.ID))
	if err != nil {
		t.Fatalf("おすすめフォローの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var got []seedRole
	var previous time.Time
	for rows.Next() {
		var targetID uuid.UUID
		var createdAt, joinedAt, suggestedAt time.Time

		if err := rows.Scan(&targetID, &createdAt, &joinedAt, &suggestedAt); err != nil {
			t.Fatalf("おすすめフォローの読み取りに失敗: %v", err)
		}

		role := roles[model.ProfileID(targetID)]
		target, err := accountForRole(accounts, role)
		if err != nil {
			t.Fatalf("役割 %s のアカウントの取得に失敗: %v", role, err)
		}

		if createdAt.After(storedInstant(now)) {
			t.Errorf("役割 %s の created_at = %v, want %v 以前", role, createdAt, storedInstant(now))
		}
		if !createdAt.Equal(target.profile.CreatedAt) {
			t.Errorf("役割 %s の created_at = %v, want モデルと一致 (%v)", role, createdAt, target.profile.CreatedAt)
		}
		if !joinedAt.Equal(target.profile.JoinedAt) {
			t.Errorf("役割 %s の joined_at = %v, want %v", role, joinedAt, target.profile.JoinedAt)
		}
		if !suggestedAt.Equal(storedInstant(now)) {
			t.Errorf("役割 %s のおすすめ作成日時 = %v, want %v", role, suggestedAt, storedInstant(now))
		}

		if !previous.IsZero() && !createdAt.Before(previous) {
			t.Errorf("プロフィールの created_at %v が直前の %v より古くない", createdAt, previous)
		}
		previous = createdAt

		got = append(got, role)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("おすすめフォローの取得に失敗: %v", err)
	}

	if !slices.Equal(got, suggestedFollowTargets) {
		t.Errorf("おすすめフォローの並び = %v, want %v", got, suggestedFollowTargets)
	}

	if len(readHomeTimeline(t, ctx, tx, source.profile.ID)) != 0 {
		t.Errorf("役割 %s のホームタイムラインが空でない", suggestedFollowSource)
	}
}

// rolesByProfileID maps each created profile back to the role it was created
// for, so that a row read out of the database can be reported by role.
//
// [Ja] rolesByProfileID は、作成された各プロフィールを、それが作成された元の役割へ
// 対応付ける。データベースから読み戻した行を役割で報告できるようにするため。
func rolesByProfileID(t *testing.T, accounts []seedAccount) map[model.ProfileID]seedRole {
	t.Helper()

	roles := make(map[model.ProfileID]seedRole, len(accounts))
	for _, account := range accounts {
		roles[account.profile.ID] = account.roster.role
	}

	return roles
}
