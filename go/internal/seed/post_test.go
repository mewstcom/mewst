package seed

import (
	"context"
	"database/sql"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// testAmounts is what a test asks for in place of defaultAmounts. The counts
// are a fraction of a run's, but not an arbitrarily small one: roleMain's has
// to stay above the number of months it is spread over, because a month
// without posts cannot show that the months were given different shares, which
// is the property the export's table of contents is read for.
//
// [Ja] testAmounts は、テストが defaultAmounts の代わりに求める件数。実行 1 回分の
// ごく一部だが、いくらでも小さくてよいわけではない。roleMain の件数は、それが広がる
// 月数を上回っている必要がある。ポストの無い月は、月ごとに異なる取り分が与えられて
// いることを示せず、それこそがエクスポートの目次を読んで確かめたい性質であるため。
var testAmounts = amounts{
	mainPosts:            120,
	followerPosts:        18,
	englishPosts:         9,
	discardedPosts:       4,
	followerToMainStamps: 8,
	mainToFollowerStamps: 3,
	newcomerToMainStamps: 2,
}

// TestCreatePosts verifies that every role holding posts gets the number it
// was asked for, spread over the months it is meant to reach, and that the
// profile records the newest of them.
//
// [Ja] TestCreatePosts は、ポストを持つそれぞれの役割が、求められた件数を、届くべき
// 月の範囲に散らして得ること、そしてプロフィールがその中で最も新しいものを記録する
// ことを検証する。
func TestCreatePosts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	roster := newTestRoster(t)

	// The instant is truncated to what the column can hold, so that a
	// published_at read back out of the database can be compared with the one
	// that was written rather than with a value rounded away from it.
	//
	// [Ja] 時点はカラムが保持できる精度へ切り詰める。データベースから読み戻した
	// published_at を、そこから丸められた値ではなく、書き込んだ値と比較できるように
	// するため。
	now := time.Now().Truncate(time.Microsecond)

	accounts, err := createAccounts(ctx, tx, roster, now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	// The application is built rather than created by createOauthApplication:
	// the post generator only needs an application to attribute posts to, and
	// the builder gives each call a uid of its own, which keeps this test off
	// the unique index that the mewst-web row's own test has to serialize on.
	//
	// [Ja] アプリケーションは createOauthApplication ではなくビルダーで作る。ポストの
	// 生成器が必要とするのは帰属先のアプリケーションだけであり、ビルダーは呼び出し
	// ごとに固有の uid を与える。これにより、mewst-web の行そのもののテストが直列化
	// せざるを得ない一意インデックスから、このテストは離れていられる。
	applicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()

	if err := createPosts(ctx, tx, testAmounts, applicationID, accounts, now); err != nil {
		t.Fatalf("ポストの作成に失敗: %v", err)
	}

	tests := []struct {
		role      seedRole
		wantCount int
		// wantMonths is how many distinct calendar months the posts fall
		// into, counted in the account's own time zone.
		//
		// [Ja] wantMonths は、ポストがいくつの暦月に分かれるか。アカウント自身の
		// タイムゾーンで数える。
		wantMonths int
		// wantDiscarded is how many of the posts have been deleted. They are
		// written and counted like the rest, but a screen cannot reach them.
		//
		// [Ja] wantDiscarded は、ポストのうち削除済みのものの件数。他と同じように
		// 書き込まれ、同じように数えられるが、画面からは辿り着けない。
		wantDiscarded int
	}{
		{
			// roleMain also holds the posts written for the export's output
			// contract. They are placed among its everyday posts rather than
			// into months of their own, so they add to the count without
			// adding to the months.
			//
			// [Ja] roleMain はエクスポートの出力契約のために書くポストも持つ。
			// これらは専用の月ではなく日常ポストの間へ配置されるため、件数には
			// 加わるが月数には加わらない。
			role:          roleMain,
			wantCount:     testAmounts.mainPosts + len(postVariations),
			wantMonths:    historyMonths,
			wantDiscarded: discardedVariationCount(),
		},
		{role: roleFollower, wantCount: testAmounts.followerPosts, wantMonths: followerPostMonths},
		{role: roleEnglish, wantCount: testAmounts.englishPosts, wantMonths: englishPostMonths},
		{role: roleDiscarded, wantCount: testAmounts.discardedPosts, wantMonths: discardedPostMonths},
		{
			// The newcomer holds nothing on purpose: an empty export and a
			// profile page with nothing under it can only be looked at from an
			// account that has written nothing.
			//
			// [Ja] newcomer が何も持たないのは意図的である。空のエクスポートと、
			// 下に何も並ばないプロフィール画面は、何も書いていないアカウントから
			// しか見られない。
			role: roleNewcomer,
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			account, err := accountForRole(accounts, tt.role)
			if err != nil {
				t.Fatalf("役割 %s のアカウントの取得に失敗: %v", tt.role, err)
			}

			posts := readPosts(t, ctx, tx, account)
			if len(posts) != tt.wantCount {
				t.Fatalf("ポスト数 = %d, want %d", len(posts), tt.wantCount)
			}

			assertLastPostAt(t, ctx, tx, account, posts)
			// The navigation is checked only where a screen could reach
			// it. assertPostNavigation filters the way PostRecord.kept
			// does, which drops the posts of a discarded profile, so the
			// discarded role would be compared against no rows at all.
			//
			// [Ja] 前後リンクの検証は、画面から辿り着ける場合にだけ行う。
			// assertPostNavigation は PostRecord.kept と同じ絞り込みをするため、
			// 削除済みプロフィールのポストは対象から外れ、discarded 役割では
			// 1 行も比較されないことになる。
			if account.profile.DiscardedAt == nil {
				assertPostNavigation(t, ctx, tx, account.profile.ID, tt.wantCount-tt.wantDiscarded)
			}

			if len(posts) == 0 {
				return
			}

			assertPostsAreReachable(t, account, posts, applicationID, now)

			monthCounts := postsPerMonth(t, account, posts)
			if len(monthCounts) != tt.wantMonths {
				t.Errorf("ポストが分かれた月数 = %d, want %d", len(monthCounts), tt.wantMonths)
			}
		})
	}

	// The months are given different shares of the posts. A table of contents
	// where every month holds the same number cannot be told apart from one
	// that counted once and repeated the answer, which is the mistake those
	// numbers are read to catch.
	//
	// [Ja] 月には異なる取り分が与えられる。どの月も同じ件数の目次は、1 度数えた
	// 答えを繰り返しているだけの目次と区別できない。それこそが、あの数字を読んで
	// 捕まえたい誤りである。
	main, err := accountForRole(accounts, roleMain)
	if err != nil {
		t.Fatalf("役割 %s のアカウントの取得に失敗: %v", roleMain, err)
	}

	counts := postsPerMonth(t, main, readPosts(t, ctx, tx, main))
	if len(slices.Compact(slices.Sorted(maps.Values(counts)))) < 2 {
		t.Errorf("main の月ごとの件数 = %v, want 月によって異なる件数", counts)
	}
}

// seededPost is one post read back out of the database.
//
// [Ja] seededPost は、データベースから読み戻したポスト 1 件。
type seededPost struct {
	content            string
	publishedAt        time.Time
	oauthApplicationID model.OauthApplicationID
}

// readPosts returns the account's posts, oldest first.
//
// [Ja] readPosts は、アカウントのポストを古いものから順に返す。
func readPosts(t *testing.T, ctx context.Context, tx *sql.Tx, account seedAccount) []seededPost {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT content, published_at, oauth_application_id
		FROM posts
		WHERE profile_id = $1
		ORDER BY published_at
	`, uuid.UUID(account.profile.ID))
	if err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var posts []seededPost
	for rows.Next() {
		var (
			post          seededPost
			applicationID uuid.UUID
		)
		if err := rows.Scan(&post.content, &post.publishedAt, &applicationID); err != nil {
			t.Fatalf("ポストの読み取りに失敗: %v", err)
		}
		post.oauthApplicationID = model.OauthApplicationID(applicationID)
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}

	return posts
}

// assertPostsAreReachable checks that every post is one a screen can arrive
// at: attributed to an application, written by an account that had already
// joined, and published by the time the run happened.
//
// [Ja] assertPostsAreReachable は、すべてのポストが画面から辿り着けるものであること
// を確認する。アプリケーションに帰属し、すでに参加していたアカウントによって書かれ、
// 実行が行われた時点までに公開されていること。
func assertPostsAreReachable(
	t *testing.T,
	account seedAccount,
	posts []seededPost,
	applicationID model.OauthApplicationID,
	now time.Time,
) {
	t.Helper()

	joinedAt := account.profile.JoinedAt

	for _, post := range posts {
		if post.content == "" {
			t.Error("本文が空のポストがある")
		}
		if post.oauthApplicationID != applicationID {
			t.Errorf("oauth_application_id = %s, want %s", post.oauthApplicationID, applicationID)
		}
		if !post.publishedAt.After(joinedAt) {
			t.Errorf("published_at = %v, want joined_at (%v) より後", post.publishedAt, joinedAt)
		}
		if post.publishedAt.After(now) {
			t.Errorf("published_at = %v, want 実行時点 (%v) 以前", post.publishedAt, now)
		}
	}
}

// assertLastPostAt checks that the profile records the newest post, and
// records nothing when there is none.
//
// [Ja] assertLastPostAt は、プロフィールが最も新しいポストを記録していること、
// ポストが無ければ何も記録していないことを確認する。
func assertLastPostAt(t *testing.T, ctx context.Context, tx *sql.Tx, account seedAccount, posts []seededPost) {
	t.Helper()

	var lastPostAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
		SELECT last_post_at FROM profiles WHERE id = $1
	`, uuid.UUID(account.profile.ID)).Scan(&lastPostAt); err != nil {
		t.Fatalf("プロフィールの last_post_at の取得に失敗: %v", err)
	}

	if len(posts) == 0 {
		if lastPostAt.Valid {
			t.Errorf("last_post_at = %v, want NULL", lastPostAt.Time)
		}
		if account.profile.LastPostAt != nil {
			t.Errorf("モデルの LastPostAt = %v, want nil", account.profile.LastPostAt)
		}

		return
	}

	newest := posts[len(posts)-1].publishedAt
	if !lastPostAt.Valid || !lastPostAt.Time.Equal(newest) {
		t.Errorf("last_post_at = %v, want %v", lastPostAt, newest)
	}

	// The model travels with the row, so that a generator that receives this
	// account afterwards is not told that nothing has been written under it.
	//
	// [Ja] モデルは行と一緒に持ち回る。この後にこのアカウントを受け取る生成器へ、
	// まだ何も書かれていないと告げないようにするため。
	if account.profile.LastPostAt == nil || !account.profile.LastPostAt.Equal(newest) {
		t.Errorf("モデルの LastPostAt = %v, want %v", account.profile.LastPostAt, newest)
	}
}

// postsPerMonth counts the posts of each calendar month, in the account's own
// time zone. The zone is what decides which month a post belongs to, and the
// export splits its archive on that boundary.
//
// [Ja] postsPerMonth は、暦月ごとのポストの件数を、アカウント自身のタイムゾーンで
// 数える。ポストがどの月に属するのかを決めるのはタイムゾーンであり、エクスポートは
// その境界でアーカイブを分割する。
func postsPerMonth(t *testing.T, account seedAccount, posts []seededPost) map[string]int {
	t.Helper()

	location, err := time.LoadLocation(account.user.TimeZone)
	if err != nil {
		t.Fatalf("タイムゾーン %q の読み込みに失敗: %v", account.user.TimeZone, err)
	}

	counts := make(map[string]int)
	for _, post := range posts {
		counts[post.publishedAt.In(location).Format("2006-01")]++
	}

	return counts
}

// TestMonthlyPostCounts verifies that the share-out reaches the total it was
// given, gives every month a share, and does not give them all the same one.
//
// [Ja] TestMonthlyPostCounts は、配分が与えられた合計に到達すること、すべての月に
// 取り分を与えること、そしてそのすべてを同じにしないことを検証する。
func TestMonthlyPostCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		total  int
		months int
		// wantEveryMonth says whether every month is expected to hold at
		// least one post. It only holds where the total clears the months.
		//
		// [Ja] wantEveryMonth は、すべての月が 1 件以上を持つと期待されるか。
		// 合計が月数を上回っている場合にだけ成り立つ。
		wantEveryMonth bool
		// wantVarying says whether the months are expected to differ from
		// each other.
		//
		// [Ja] wantVarying は、月どうしが互いに異なると期待されるか。
		wantVarying bool
	}{
		{
			name:           "実行 1 回分の件数は 36 か月すべてに異なる取り分を与える",
			total:          defaultAmounts.mainPosts,
			months:         historyMonths,
			wantEveryMonth: true,
			wantVarying:    true,
		},
		{
			name:           "テストの件数でも月ごとの差は残る",
			total:          testAmounts.mainPosts,
			months:         historyMonths,
			wantEveryMonth: true,
			wantVarying:    true,
		},
		{
			name:           "半年ぶんの件数も月へ配分される",
			total:          defaultAmounts.followerPosts,
			months:         followerPostMonths,
			wantEveryMonth: true,
			wantVarying:    true,
		},
		{
			name:           "数件でも月をまたいで配分される",
			total:          defaultAmounts.discardedPosts,
			months:         discardedPostMonths,
			wantEveryMonth: true,
		},
		{
			name:   "1 か月しか無ければすべてがその月に入る",
			total:  10,
			months: 1,
		},
		{
			// Zero is a boundary value for the allocation function itself.
			//
			// [Ja] 配分関数自体の境界値として 0 件を検証する。
			name:   "0 件なら月はすべて空になる",
			total:  0,
			months: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			counts := monthlyPostCounts(tt.total, tt.months)

			if len(counts) != tt.months {
				t.Fatalf("月数 = %d, want %d", len(counts), tt.months)
			}

			total := 0
			for _, count := range counts {
				if count < 0 {
					t.Errorf("月ごとの件数 = %v, want 負の件数を含まない", counts)
				}
				total += count
			}
			if total != tt.total {
				t.Errorf("合計 = %d, want %d (内訳 %v)", total, tt.total, counts)
			}

			if tt.wantEveryMonth && slices.Contains(counts, 0) {
				t.Errorf("月ごとの件数 = %v, want すべての月が 1 件以上", counts)
			}

			if tt.wantVarying && len(slices.Compact(slices.Sorted(slices.Values(counts)))) < 2 {
				t.Errorf("月ごとの件数 = %v, want 月によって異なる件数", counts)
			}
		})
	}
}

// TestMonthlyPostCountsWithoutMonths verifies that a span of no months is
// answered with no months, rather than with a panic on a negative length.
//
// [Ja] TestMonthlyPostCountsWithoutMonths は、月が 1 つも無い期間に対して、負の
// 長さによるパニックではなく、月の無い答えが返ることを検証する。
func TestMonthlyPostCountsWithoutMonths(t *testing.T) {
	t.Parallel()

	for _, months := range []int{0, -1} {
		if counts := monthlyPostCounts(10, months); len(counts) != 0 {
			t.Errorf("monthlyPostCounts(10, %d) = %v, want 空", months, counts)
		}
	}
}

// TestMonthWindow verifies that the windows a role's posts are placed in cover
// the months back to back, are counted in the account's own time zone, and
// stop at the instant the run is anchored to.
//
// [Ja] TestMonthWindow は、ある役割のポストが置かれる区間が、月を隙間なく覆うこと、
// アカウント自身のタイムゾーンで数えられること、そして実行が基準とする時点で
// 止まることを検証する。
func TestMonthWindow(t *testing.T) {
	t.Parallel()

	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatalf("タイムゾーンの読み込みに失敗: %v", err)
	}

	const months = 4

	// The instant is one where the two zones disagree about the date: the
	// evening of 31 March in UTC is already 1 April in Tokyo. A window built
	// from the wrong zone therefore lands on a different month here, which is
	// the mistake this test is for.
	//
	// [Ja] 時点は、2 つのタイムゾーンで日付が食い違うものを選んでいる。UTC の
	// 3 月 31 日の夕方は、東京ではすでに 4 月 1 日である。誤ったタイムゾーンで
	// 組み立てた区間はここで別の月へ落ちることになり、それがこのテストの対象とする
	// 誤りである。
	now := time.Date(2026, time.March, 31, 16, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		location  *time.Location
		wantFirst time.Time
	}{
		{
			name:      "UTC では 3 月が最後の月になる",
			location:  time.UTC,
			wantFirst: time.Date(2025, time.December, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			// 16:00 UTC on 31 March is one in the morning of 1 April in
			// Tokyo, so the month the run happens in — and every window
			// before it — moves forward by one.
			//
			// [Ja] 3 月 31 日 16:00 UTC は、東京では 4 月 1 日の 1 時である。
			// 実行が行われている月と、それ以前のすべての区間が 1 つ先へ動く。
			name:      "東京では 4 月が最後の月になる",
			location:  tokyo,
			wantFirst: time.Date(2026, time.January, 1, 0, 0, 0, 0, tokyo),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var previousEnd time.Time
			for index := range months {
				start, end := monthWindow(now, tt.location, months, index)

				switch index {
				case 0:
					if !start.Equal(tt.wantFirst) {
						t.Errorf("最も古い月の開始 = %v, want %v", start, tt.wantFirst)
					}
				default:
					if !start.Equal(previousEnd) {
						t.Errorf("%d 番目の月の開始 = %v, want 前の月の終わり %v", index, start, previousEnd)
					}
				}

				if !end.After(start) {
					t.Errorf("%d 番目の月 = [%v, %v), want 幅のある区間", index, start, end)
				}

				previousEnd = end
			}

			// The month the run happens in is still going, so the last window
			// reaches up to the run and no further. A window that ran to the
			// end of the month would hold posts that have not happened yet.
			//
			// [Ja] 実行が行われている月はまだ続いているため、最後の区間は実行まで
			// で止まり、その先へは伸びない。月末まで伸びる区間は、まだ起きていない
			// ポストを持つことになる。
			if !previousEnd.Equal(now) {
				t.Errorf("最も新しい月の終わり = %v, want 実行時点 %v", previousEnd, now)
			}
		})
	}
}

// TestPostTimeInWindow verifies that a month's posts are spread inside it, in
// order, without any of them landing on either edge of the month.
//
// [Ja] TestPostTimeInWindow は、ある月のポストがその内側へ順に散らされること、
// そしてそのいずれも月の両端に載らないことを検証する。
func TestPostTimeInWindow(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	// Use the largest monthly count at the default volume to verify that
	// those posts remain ordered and fit inside the month.
	//
	// [Ja] 既定の生成量での最大月件数を使い、ポストが順序を保って月の内側に収まる
	// ことを検証する。
	count := slices.Max(monthlyPostCounts(defaultAmounts.mainPosts, historyMonths))

	previous := start
	for index := range count {
		at := postTimeInWindow(start, end, index, count)

		if !at.After(start) || !at.Before(end) {
			t.Fatalf("%d 番目の時点 = %v, want [%v, %v) の内側", index, at, start, end)
		}
		if index > 0 && !at.After(previous) {
			t.Fatalf("%d 番目の時点 = %v, want 前の時点 %v より後", index, at, previous)
		}

		previous = at
	}
}

// TestPostWriter_WritesNothingWithoutMonths verifies that a spec with posts
// but no months to spread them over writes no posts and leaves last_post_at
// unset, rather than recording an instant no post was published at.
//
// [Ja] TestPostWriter_WritesNothingWithoutMonths は、ポストの件数を持ちながら
// それを散らす月を持たない spec が、ポストを 1 件も書かず、last_post_at も未設定
// のままにすることを検証する。どのポストも公開されていない時点を記録してしまわない
// ようにするため。
func TestPostWriter_WritesNothingWithoutMonths(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	account := seedAccount{
		user:    &model.User{TimeZone: "Asia/Tokyo"},
		profile: &model.Profile{ID: testutil.NewProfileBuilder(t, tx).Build()},
	}
	writer := newPostWriter(
		tx,
		testutil.NewOauthApplicationBuilder(t, tx).Build(),
		time.Now().Truncate(time.Microsecond),
	)

	spec := rolePostSpec{role: roleMain, count: 10, months: 0, bodies: mainPostBodies}
	if err := writer.writeRolePosts(ctx, account, spec); err != nil {
		t.Fatalf("ポストの作成に失敗: %v", err)
	}

	posts := readPosts(t, ctx, tx, account)
	if len(posts) != 0 {
		t.Errorf("ポスト数 = %d, want 0", len(posts))
	}
	assertLastPostAt(t, ctx, tx, account, posts)
}

// TestPostWriter_NavigationFollowsPublicationOrder checks previous/next links
// for posts inserted out of chronological order, including timestamps within
// the same millisecond and ties at the database's microsecond precision.
//
// [Ja] TestPostWriter_NavigationFollowsPublicationOrder は、日時順と異なる順序で
// 挿入されたポストの前後リンクを検証する。同じミリ秒内の日時と、DBのマイクロ秒精度で
// 同じ日時になるポストも含める。
func TestPostWriter_NavigationFollowsPublicationOrder(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.February, 28, 23, 59, 59, 999000000, time.UTC)
	tests := []struct {
		name  string
		times []time.Time
	}{
		{
			name:  "月を遡って挿入する",
			times: []time.Time{base, base.AddDate(0, -1, 0), base.AddDate(0, -2, 0)},
		},
		{
			name: "同じミリ秒内と次のミリ秒のポストを逆順に挿入する",
			times: []time.Time{
				base.Add(time.Millisecond),
				base.Add(999 * time.Microsecond),
				base.Add(2 * time.Microsecond),
				base.Add(time.Microsecond),
				base,
			},
		},
		{
			name:  "同じ公開日時のポストはIDで順序が決まる",
			times: []time.Time{base, base, base},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := context.Background()
			profileID := testutil.NewProfileBuilder(t, tx).Build()
			account := seedAccount{profile: &model.Profile{ID: profileID}}
			writer := &postWriter{
				tx:            tx,
				posts:         repository.NewPostRepository(query.New(tx)),
				applicationID: testutil.NewOauthApplicationBuilder(t, tx).Build(),
			}

			for _, publishedAt := range tt.times {
				if _, err := writer.writePost(ctx, account, "前後のポストの確認", publishedAt); err != nil {
					t.Fatalf("ポストの作成に失敗: %v", err)
				}
			}

			assertPostNavigation(t, ctx, tx, profileID, len(tt.times))
		})
	}
}

// assertPostNavigation compares Rails' ID-filtered previous/next lookups with
// each post's neighbors in publication order, including the missing neighbor
// at either end. The kept scopes match PostRecord.kept.
//
// [Ja] assertPostNavigation は、IDで絞り込むRailsの前後検索と、公開日時順での
// 隣接ポストを比較する。先頭・末尾で該当する隣接ポストが存在しないことも確認する。
// 削除済みの除外条件は PostRecord.kept に合わせる。
func assertPostNavigation(t *testing.T, ctx context.Context, tx *sql.Tx, profileID model.ProfileID, wantCount int) {
	t.Helper()

	var count, mismatches int
	if err := tx.QueryRowContext(ctx, `
		WITH profile_posts AS (
			SELECT posts.id, posts.published_at
			FROM posts
			JOIN profiles ON profiles.id = posts.profile_id
			WHERE posts.profile_id = $1
			  AND posts.discarded_at IS NULL
			  AND profiles.discarded_at IS NULL
		), ordered AS (
			SELECT id,
				LAG(id) OVER (ORDER BY published_at, id) AS previous_id,
				LEAD(id) OVER (ORDER BY published_at, id) AS next_id
			FROM profile_posts
		)
		SELECT COUNT(*), COUNT(*) FILTER (
			WHERE previous_id IS DISTINCT FROM (
				SELECT previous.id FROM profile_posts previous
				WHERE previous.id < ordered.id
				ORDER BY previous.published_at DESC, previous.id DESC LIMIT 1
			) OR next_id IS DISTINCT FROM (
				SELECT next.id FROM profile_posts next
				WHERE next.id > ordered.id
				ORDER BY next.published_at, next.id LIMIT 1
			)
		)
		FROM ordered
	`, uuid.UUID(profileID)).Scan(&count, &mismatches); err != nil {
		t.Fatalf("前後のポストの検証に失敗: %v", err)
	}
	if count != wantCount {
		t.Errorf("前後リンクを検証したポスト数 = %d, want %d", count, wantCount)
	}
	if mismatches != 0 {
		t.Errorf("前後のポストが公開日時順と一致しない件数 = %d, want 0", mismatches)
	}
}
