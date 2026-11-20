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

// followPairは、フォローの構成のうち1つの辺を、その両端の役割で表したもの。
type followPair struct {
	source seedRole
	target seedRole
}

// storedFollowは、データベースから読み戻したフォロー1件。
type storedFollow struct {
	followedAt time.Time
	createdAt  time.Time
}

// timelineSourcePostは、テストが読み戻したプロフィールのポスト1件と、
// それが画面から辿り着ける状態にあるかどうか。
type timelineSourcePost struct {
	id          model.PostID
	publishedAt time.Time
	discarded   bool
}

// TestFollowEdgesは、フォローの構成がアプリケーションによって残されえた
// ものであること、そして対称ではないことを検証する。
func TestFollowEdges(t *testing.T) {
	t.Parallel()

	seen := make(map[followPair]bool, len(followEdges))
	for _, edge := range followEdges {
		if edge.source == edge.target {
			t.Errorf("役割%sが自分自身をフォローしている", edge.source)
		}

		if !slices.Contains(allSeedRoles, edge.source) || !slices.Contains(allSeedRoles, edge.target) {
			t.Errorf("フォロー%s -> %sに名簿の持たない役割が含まれる", edge.source, edge.target)
		}

		// このカラムの組には一意インデックスがあるため、重複する辺は、
		// その2度目の書き込みが作れない行になる。
		pair := followPair{source: edge.source, target: edge.target}
		if seen[pair] {
			t.Errorf("フォロー%s -> %sが重複している", edge.source, edge.target)
		}
		seen[pair] = true

		if edge.monthsAgo <= 0 {
			t.Errorf("フォロー%s -> %sのmonthsAgo = %d、期待値 = 1以上", edge.source, edge.target, edge.monthsAgo)
		}
	}

	// プロフィールは、フォローしている数とフォローされている数を示す。
	// すべてのフォローが返される構成では、この2つの数は誰にとっても同じであり、
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

	// おすすめが提示されるアカウントは、フォローを持たないものである必要が
	// ある。それを提示する画面は、アカウントがフォローを1つも持たないうちに見る
	// ものであるため。
	for _, edge := range followEdges {
		if edge.source == suggestedFollowSource {
			t.Errorf("おすすめの提示先である役割%sが%sをフォローしている", edge.source, edge.target)
		}
	}
}

// TestSuggestedFollowTargetsは、おすすめのいずれもが画面に表示されるもので
// あることを検証する。
func TestSuggestedFollowTargets(t *testing.T) {
	t.Parallel()

	seen := make(map[seedRole]bool, len(suggestedFollowTargets))
	for _, role := range suggestedFollowTargets {
		if seen[role] {
			t.Errorf("おすすめの役割%sが重複している", role)
		}
		seen[role] = true

		if role == suggestedFollowSource {
			t.Errorf("役割%sが自分自身をおすすめされている", role)
		}

		// 画面はおすすめを、まだ残っているプロフィールへ絞り込む。削除済みの
		// ものは、何にも表示されない行になる。
		if role == roleDiscarded {
			t.Errorf("削除済みプロフィールの役割%sがおすすめに含まれている", role)
		}

		if !slices.Contains(allSeedRoles, role) {
			t.Errorf("おすすめに名簿の持たない役割%sが含まれる", role)
		}
	}
}

// TestCreateFollowsは、フォローの構成が書かれたとおりにデータベースへ届く
// こと、各プロフィールのホームタイムラインがそのフォローの与えたものを持つこと、
// そしてフォローを持たないアカウントには代わりにおすすめが提示されることを検証する。
func TestCreateFollows(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 時点はカラムが保持できる精度へ切り詰める。データベースから読み戻した
	// followed_atを、そこから丸められた値ではなく、書き込んだ値と比較できるように
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

// assertFollowRowsは、データベースのフォローをフォローの構成と突き合わせる。
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
		t.Fatalf("フォローの件数 = %d、期待値 = %d", len(got), len(followEdges))
	}

	for _, edge := range followEdges {
		pair := followPair{source: edge.source, target: edge.target}

		follow, ok := got[pair]
		if !ok {
			t.Errorf("フォロー%s -> %sが作成されていない", edge.source, edge.target)
			continue
		}

		want := storedInstant(now.AddDate(0, -edge.monthsAgo, 0))
		if !follow.followedAt.Equal(want) {
			t.Errorf("フォロー%s -> %sのfollowed_at = %v、期待値 = %v", edge.source, edge.target, follow.followedAt, want)
		}

		// フォローは、それが行われた瞬間に作成される。フォローを並べる画面は
		// created_atを読むため、行われた時点と違う時点で作成された行は、
		// followed_atが否定する順序で並べられることになる。
		if !follow.createdAt.Equal(want) {
			t.Errorf("フォロー%s -> %sのcreated_at = %v、期待値 = %v", edge.source, edge.target, follow.createdAt, want)
		}
	}
}

// assertHomeTimelinesは、各プロフィールのホームタイムラインを、自身のポストと
// そのフォローが与えたものと突き合わせる。
//
// 期待する内容は、生成器がデータベースへ投げた問いを投げ直すのではなく、書き込まれた
// ポストからGoの側で組み立てる。それを埋めたクエリと突き合わせたタイムラインは、
// そのクエリが何を言おうと自分自身と一致してしまう。
func assertHomeTimelines(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount, now time.Time) {
	t.Helper()

	// 取り込みの上限が効くのは、相手がそれを超える件数のポストを既に持って
	// いた場合だけである。期待値を組み立てながら数え、あとで確認する。testAmountsを
	// 小さくしたときに、テストがその分岐へ届かなくなるのではなく、ここで失敗する
	// ようにするため。
	capped := 0

	// 誰もフォローしていないアカウントも、自身のポストはホーム画面に持つ。
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
				t.Fatalf("役割%sのアカウントの取得に失敗: %v", edge.target, err)
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
			t.Errorf("役割%sのホームタイムラインの件数 = %d、期待値 = %d", account.roster.role, len(got), len(want))
		}

		for id, publishedAt := range want {
			stored, ok := got[id]
			if !ok {
				t.Errorf("役割%sのホームタイムラインにポスト%sが無い", account.roster.role, uuid.UUID(id))
				continue
			}

			// タイムラインは、ポストを参照せずに並べ替えられるよう公開日時を
			// 持つ。別の時点を持つ行は、そのポストをプロフィールが置いていない位置へ
			// 置くことになる。
			if !stored.Equal(publishedAt) {
				t.Errorf("役割%sのホームタイムラインのポスト%sのpublished_at = %v、期待値 = %v",
					account.roster.role, uuid.UUID(id), stored, publishedAt)
			}
		}

		for id := range got {
			if _, ok := want[id]; !ok {
				t.Errorf("役割%sのホームタイムラインに余分なポスト%sがある", account.roster.role, uuid.UUID(id))
			}
		}
	}

	if capped == 0 {
		t.Errorf("既にあったポストの取り込みが上限 (%d件) に達するフォローが無い", backfilledTimelinePosts)
	}
	if ownOnly == 0 {
		t.Error("フォローを持たないまま自身のポストがホームタイムラインに並ぶアカウントが無い")
	}
}

// wantOwnTimelinePostsは、あるアカウントが自身のホームタイムラインへ置いたはず
// のものを返す。そのアカウントが持つポストのうち、画面から辿り着けるものになる。
func wantOwnTimelinePosts(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	account seedAccount,
) map[model.PostID]time.Time {
	t.Helper()

	own := make(map[model.PostID]time.Time)

	// 削除済みプロフィールでサインインする人はいない。その自身のホーム画面は
	// どの実行も開かないものであり、そのための行も書かれない。
	if account.profile.DiscardedAt != nil {
		return own
	}

	for _, post := range readTimelineSourcePosts(t, ctx, tx, account.profile.ID) {
		// 削除されたポストはタイムラインの行も一緒に取り除く。書いた本人の分も
		// そこに含まれる。
		if post.discarded {
			continue
		}

		own[post.id] = post.publishedAt
	}

	return own
}

// wantTimelinePostsは、followedAtに行われたフォローがフォローした側の
// タイムラインへ入れたはずのものを返す。相手がそれ以降に公開したもののすべてと、
// それより前に公開していたもののうち新しいbackfilledTimelinePosts件になる。
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

	// 削除済みプロフィールのポストは表示されないため、そのプロフィールへの
	// フォローはフォローした側へ何も与えない。今日そうしたプロフィールを指す辺は
	// 無く、それを件数から読み取らせるのではなく、ここで述べる。
	if target.profile.DiscardedAt != nil {
		return delivered, backfilled, 0
	}

	var older []timelineSourcePost
	for _, post := range readTimelineSourcePosts(t, ctx, tx, target.profile.ID) {
		// 削除されたポストはタイムラインの行も一緒に取り除くため、フォローが
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

	// ポストは古いものから返るため、それより前のもののうち新しいものは、その
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

// readTimelineSourcePostsは、プロフィールのポストを古いものから順に返す。
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

// readHomeTimelineは、プロフィールのホームタイムラインをポストごとに返す。
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

// assertSuggestedFollowsは、おすすめがフォローを持たないアカウントへ、画面が
// 並べる順序で提示されることを検証する。
func assertSuggestedFollows(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount, now time.Time) {
	t.Helper()

	roles := rolesByProfileID(t, accounts)

	source, err := accountForRole(accounts, suggestedFollowSource)
	if err != nil {
		t.Fatalf("役割%sのアカウントの取得に失敗: %v", suggestedFollowSource, err)
	}

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM suggested_follows`).Scan(&count); err != nil {
		t.Fatalf("おすすめフォローの件数の取得に失敗: %v", err)
	}
	if count != len(suggestedFollowTargets) {
		t.Errorf("おすすめフォローの件数 = %d、期待値 = %d", count, len(suggestedFollowTargets))
	}

	// Search::ShowControllerのプロフィール関連と同じ絞り込み・ソートキーを
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
			t.Fatalf("役割%sのアカウントの取得に失敗: %v", role, err)
		}

		if createdAt.After(storedInstant(now)) {
			t.Errorf("役割%sのcreated_at = %v、期待値 = %v以前", role, createdAt, storedInstant(now))
		}
		if !createdAt.Equal(target.profile.CreatedAt) {
			t.Errorf("役割%sのcreated_at = %v、期待値 = モデルと一致 (%v)", role, createdAt, target.profile.CreatedAt)
		}
		if !joinedAt.Equal(target.profile.JoinedAt) {
			t.Errorf("役割%sのjoined_at = %v、期待値 = %v", role, joinedAt, target.profile.JoinedAt)
		}
		if !suggestedAt.Equal(storedInstant(now)) {
			t.Errorf("役割%sのおすすめ作成日時 = %v、期待値 = %v", role, suggestedAt, storedInstant(now))
		}

		if !previous.IsZero() && !createdAt.Before(previous) {
			t.Errorf("プロフィールのcreated_at %vが直前の%vより古くない", createdAt, previous)
		}
		previous = createdAt

		got = append(got, role)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("おすすめフォローの取得に失敗: %v", err)
	}

	if !slices.Equal(got, suggestedFollowTargets) {
		t.Errorf("おすすめフォローの並び = %v、期待値 = %v", got, suggestedFollowTargets)
	}

	if len(readHomeTimeline(t, ctx, tx, source.profile.ID)) != 0 {
		t.Errorf("役割%sのホームタイムラインが空でない", suggestedFollowSource)
	}
}

// rolesByProfileIDは、作成された各プロフィールを、それが作成された元の役割へ
// 対応付ける。データベースから読み戻した行を役割で報告できるようにするため。
func rolesByProfileID(t *testing.T, accounts []seedAccount) map[model.ProfileID]seedRole {
	t.Helper()

	roles := make(map[model.ProfileID]seedRole, len(accounts))
	for _, account := range accounts {
		roles[account.profile.ID] = account.roster.role
	}

	return roles
}
