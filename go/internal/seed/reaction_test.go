package seed

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// stampPairは、スタンプの向きを、その両端の役割で表したもの。
type stampPair struct {
	source seedRole
	target seedRole
}

// stampPlacementは、1つの向きが生み出すと期待されるもの。スタンプの件数と、
// その最初の1件が相手のポストのどこに座るか。
type stampPlacement struct {
	count  int
	offset int
}

// firstScreenPostsは、画面がスクロールせずに一覧するポストのおおよその件数。
//
// スタンプの配置はstampStrideだけでなくこの件数に対しても検査する。生成器が選ぶと
// 期待される位置はstampStrideから求めるため、スタンプを引き離す間隔も、一覧の先頭
// から遠ざける間隔も、それ自身と一致してしまう。間隔が存在する理由は、プロフィールを
// 開いた画面にポストカードの2つの状態がどちらも見えることであり、ここで固定するのは
// その性質になる。
const firstScreenPosts = 10

// minStampGapは、2つのスタンプ済みポストの間に未スタンプのポストが1件
// 残る最小の距離。
const minStampGap = 2

// storedStampは、データベースから読み戻したスタンプ1件と、それが押された
// ポスト。
type storedStamp struct {
	id          uuid.UUID
	postID      model.PostID
	stampedAt   time.Time
	createdAt   time.Time
	publishedAt time.Time
	pair        stampPair
}

// storedNotificationは、データベースから読み戻した通知1件。
type storedNotification struct {
	notifiableType string
	notifiableID   uuid.UUID
	notifiedAt     time.Time
	createdAt      time.Time
	pair           stampPair
}

// TestStampSpecsは、スタンプが画面に表示される先へ、各向きに与えられた件数と
// 位置で押されることを検証する。
func TestStampSpecs(t *testing.T) {
	t.Parallel()

	specs := stampSpecs(defaultAmounts)
	want := map[stampPair]stampPlacement{
		{source: roleFollower, target: roleMain}: {count: defaultAmounts.followerToMainStamps, offset: 0},
		{source: roleMain, target: roleFollower}: {count: defaultAmounts.mainToFollowerStamps, offset: 0},
		{source: roleNewcomer, target: roleMain}: {count: defaultAmounts.newcomerToMainStamps, offset: 1},
	}
	if len(specs) != len(want) {
		t.Errorf("スタンプの向きの数 = %d、期待値 = %d", len(specs), len(want))
	}

	seen := make(map[stampPair]bool, len(specs))
	for _, spec := range specs {
		if spec.source == spec.target {
			t.Errorf("役割%sが自分自身のポストへスタンプを押している", spec.source)
		}

		if !slices.Contains(allSeedRoles, spec.source) || !slices.Contains(allSeedRoles, spec.target) {
			t.Errorf("スタンプ%s -> %sに名簿の持たない役割が含まれる", spec.source, spec.target)
		}

		// 同じ役割ペアを二重に定義すると同じポスト群を再選択し、2度目の挿入で
		// (profile_id, post_id) の一意制約に違反する。
		pair := stampPair{source: spec.source, target: spec.target}
		if seen[pair] {
			t.Errorf("スタンプ%s -> %sが重複している", spec.source, spec.target)
		}
		seen[pair] = true
		placement, ok := want[pair]
		if !ok {
			t.Errorf("スタンプ%s -> %sは期待していない向き", spec.source, spec.target)
			continue
		}
		if spec.count != placement.count {
			t.Errorf("スタンプ%s -> %sの件数 = %d、期待値 = %d",
				spec.source, spec.target, spec.count, placement.count)
		}
		if spec.offset != placement.offset {
			t.Errorf("スタンプ%s -> %sのoffset = %d、期待値 = %d",
				spec.source, spec.target, spec.offset, placement.offset)
		}

		// 間隔の外のoffsetは、その範囲内のoffsetが既に覆っているポストへ
		// 着く。そのような組を与えられた2つの向きは、結局同じポストをスタンプする
		// ことになる。
		if spec.offset < 0 || spec.offset >= stampStride {
			t.Errorf("スタンプ%s -> %sのoffset = %d、期待値 = 0以上%d未満",
				spec.source, spec.target, spec.offset, stampStride)
		}

		// 削除済みプロフィールはどの画面にも表示されないため、それが押した
		// スタンプも、それが起こした通知も表示されない。
		if spec.source == roleDiscarded || spec.target == roleDiscarded {
			t.Errorf("スタンプ%s -> %sに削除済みプロフィールの役割が含まれる", spec.source, spec.target)
		}

		if spec.count <= 0 {
			t.Errorf("スタンプ%s -> %sのcount = %d、期待値 = 1以上", spec.source, spec.target, spec.count)
		}
	}

	// 同じプロフィールをスタンプする2つの向きは、別のポストを取る必要がある。
	// ポストがいつスタンプされるのかは、それがいつ公開されたのかから求まるため、同じ
	// ポストを選んだ組は、時点を共有する通知を受け取り手へ与えることになる。一覧は
	// その時点で並べる。
	type postSlot struct {
		target seedRole
		offset int
	}

	takenPosts := make(map[postSlot]bool, len(specs))
	for _, spec := range specs {
		slot := postSlot{target: spec.target, offset: spec.offset}
		if takenPosts[slot] {
			t.Errorf("役割%sのポストを、offset %dの向きが2つ以上スタンプしている",
				spec.target, spec.offset)
		}
		takenPosts[slot] = true
	}
}

// TestStampSpecs_NotificationListsは、向きの組み合わせが、開発者がサインイン
// して読む通知一覧のそれぞれを、読むために必要な状態にすることを検証する。
func TestStampSpecs_NotificationLists(t *testing.T) {
	t.Parallel()

	specs := stampSpecs(defaultAmounts)

	// 通知一覧は、roleMainとroleFollowerが作る組の両端から読まれる。片方の
	// 向きしかスタンプを押さない実行は、もう一方の端のアカウントに空の一覧を残すことに
	// なる。
	targets := make(map[seedRole]bool, len(specs))
	for _, spec := range specs {
		targets[spec.target] = true
	}
	for _, role := range []seedRole{roleMain, roleFollower} {
		if !targets[role] {
			t.Errorf("役割%sが通知を1件も受け取らない", role)
		}
	}

	// 件数は意図して非対称にしている。片方の端はページを送れるだけの長さの一覧を
	// 読み、もう一方は数件を読む。同じ件数では、一覧が持つのは持ち主が起こした通知では
	// なく受け取った通知であることを示す画面が無くなる。
	if defaultAmounts.followerToMainStamps <= defaultAmounts.mainToFollowerStamps {
		t.Errorf("followerToMainStamps = %d, mainToFollowerStamps = %d、期待値 = followerToMainStampsのほうが多いこと",
			defaultAmounts.followerToMainStamps, defaultAmounts.mainToFollowerStamps)
	}

	// 通知カードがフォローボタンを出すのは、閲覧者がフォローしていない相手の
	// ときだけである。そのためroleMainは、自身がフォローしていない役割からスタンプを
	// 押される必要がある。送り手がすべてフォロー済みであれば、そのボタンは実行が
	// 生み出すどの画面にも現れない。
	followedByMain := make(map[seedRole]bool, len(followEdges))
	for _, edge := range followEdges {
		if edge.source == roleMain {
			followedByMain[edge.target] = true
		}
	}

	unfollowed := false
	for _, spec := range specs {
		if spec.target == roleMain && !followedByMain[spec.source] {
			unfollowed = true
		}
	}
	if !unfollowed {
		t.Error("役割mainへのスタンプが、mainのフォローしていない役割から1件も無い")
	}
}

// TestStampedAtは、スタンプが、それが押されたポストより後、それを書き込んだ
// 実行より前に記録されること、そして別のポストへの2つのスタンプが同じ時点に
// 記録されないことを検証する。
func TestStampedAt(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		publishedAt time.Time
		want        time.Time
	}{
		{name: "3年前", publishedAt: now.AddDate(-3, 0, 0), want: now.AddDate(-3, 0, 0).Add(90 * time.Minute)},
		{name: "1か月前", publishedAt: now.AddDate(0, -1, 0), want: now.AddDate(0, -1, 0).Add(90 * time.Minute)},
		{name: "180分前", publishedAt: now.Add(-180 * time.Minute), want: now.Add(-90 * time.Minute)},
		{name: "120分前", publishedAt: now.Add(-120 * time.Minute), want: now.Add(-60 * time.Minute)},
		{name: "90分前", publishedAt: now.Add(-90 * time.Minute), want: now.Add(-45 * time.Minute)},
		{name: "45分前", publishedAt: now.Add(-45 * time.Minute), want: now.Add(-22*time.Minute - 30*time.Second)},
		{name: "1分前", publishedAt: now.Add(-time.Minute), want: now.Add(-30 * time.Second)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			at := stampedAt(test.publishedAt, now)
			if !at.Equal(test.want) {
				t.Errorf("stampedAt = %v、期待値 = %v", at, test.want)
			}

			if at.Before(storedInstant(test.publishedAt)) {
				t.Errorf("stampedAt = %v、期待値 = 公開日時%v以降", at, test.publishedAt)
			}
			if !at.Before(storedInstant(now)) {
				t.Errorf("stampedAt = %v、期待値 = 実行の時点%vより前", at, now)
			}
		})
	}

	// 通知一覧はnotified_atで並べるため、同じ時点を共有する2つのスタンプは、
	// シードが決めていない順序でしか並べられない組になる。
	var previous time.Time
	for _, test := range tests {
		at := stampedAt(test.publishedAt, now)

		if !previous.IsZero() && !previous.Before(at) {
			t.Errorf("公開日時%vのスタンプ日時%vが、より古いポストのスタンプ日時%vより後になっていない",
				test.publishedAt, at, previous)
		}
		previous = at
	}
}

// TestCreateReactionsは、各役割が相手のポストへ求められた件数のスタンプを、
// その新しいものの中へ間隔を空けて押すこと、そしてすべてのスタンプが、それ自身へ
// 解決できる通知を受け取り手へ渡すことを検証する。
func TestCreateReactions(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// 時点はカラムが保持できる精度へ切り詰める。データベースから読み戻した
	// stamped_atを、そこから丸められた値ではなく、書き込んだ値と比較できるように
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

	if err := createReactions(ctx, tx, testAmounts, accounts, now); err != nil {
		t.Fatalf("スタンプの作成に失敗: %v", err)
	}

	stamps := readStamps(t, ctx, tx, accounts)

	want := map[stampPair]stampPlacement{
		{source: roleFollower, target: roleMain}: {count: testAmounts.followerToMainStamps, offset: 0},
		{source: roleMain, target: roleFollower}: {count: testAmounts.mainToFollowerStamps, offset: 0},
		{source: roleNewcomer, target: roleMain}: {count: testAmounts.newcomerToMainStamps, offset: 1},
	}
	assertStampCounts(t, stamps, want)
	assertStampPlacement(t, ctx, tx, accounts, stamps, want)
	assertStampInstants(t, stamps, now)
	assertStampNotifications(t, ctx, tx, accounts, stamps)
}

// TestCreateReactions_InsufficientPostsは、スタンプ対象のポストが不足した実行が、
// 黙ってスタンプと通知を減らすのではなくエラーを報告することを検証する。
func TestCreateReactions_InsufficientPosts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		posts   int
		wantErr string
	}{
		{name: "ポストなし", posts: 0, wantErr: "2件必要ですが、0件しかありません"},
		{name: "2件に1件を選ぶと不足", posts: 2, wantErr: "2件必要ですが、1件しかありません"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, tx := testutil.SetupTx(t)
			ctx := context.Background()
			now := time.Now().Truncate(time.Microsecond)
			accounts, err := createAccounts(ctx, tx, newTestRoster(t), now)
			if err != nil {
				t.Fatalf("アカウントの作成に失敗: %v", err)
			}
			target, err := accountForRole(accounts, roleMain)
			if err != nil {
				t.Fatalf("役割mainのアカウントの取得に失敗: %v", err)
			}
			applicationID := testutil.NewOauthApplicationBuilder(t, tx).Build()
			for i := range test.posts {
				testutil.NewPostBuilder(t, tx).
					WithProfileID(target.profile.ID).
					WithOauthApplicationID(applicationID).
					WithPublishedAt(now.Add(-time.Duration(i+1) * time.Hour)).
					Build()
			}

			err = createReactions(ctx, tx, amounts{followerToMainStamps: 2}, accounts, now)
			if err == nil {
				t.Fatal("ポスト不足時にエラーを返さなかった")
			}
			for _, want := range []string{"役割followerから役割main", test.wantErr} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("エラー = %q、%qを含むことを期待", err, want)
				}
			}
			if stamps := readStamps(t, ctx, tx, accounts); len(stamps) != 0 {
				t.Errorf("不足時のスタンプ数 = %d、期待値 = 0", len(stamps))
			}
			if notifications := readStampNotifications(t, ctx, tx, accounts); len(notifications) != 0 {
				t.Errorf("不足時の通知数 = %d、期待値 = 0", len(notifications))
			}
		})
	}
}

// assertStampCountsは、データベースのスタンプを、実行が求められた件数と
// 突き合わせる。
func assertStampCounts(t *testing.T, stamps []storedStamp, want map[stampPair]stampPlacement) {
	t.Helper()

	got := make(map[stampPair]int)
	for _, stamp := range stamps {
		got[stamp.pair]++
	}

	if len(got) != len(want) {
		t.Errorf("スタンプの向きの数 = %d、期待値 = %d", len(got), len(want))
	}

	for pair, placement := range want {
		if got[pair] != placement.count {
			t.Errorf("スタンプ%s -> %sの件数 = %d、期待値 = %d", pair.source, pair.target, got[pair], placement.count)
		}
	}
}

// assertStampPlacementは、スタンプされたポストが相手の最も新しいものから、
// その向きのoffsetを起点にstampStride件に1件であること、そしてプロフィールを
// 開いた画面がポストカードの2つの状態をどちらも示す配置であることを検証する。
//
// 位置は、生成器がデータベースへ投げた問いを投げ直すのではなく、実行が書き込んだ
// ポストから求める。それを選んだクエリと突き合わせた配置は、そのクエリが何を言おうと
// 自分自身と一致してしまう。
func assertStampPlacement(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount, stamps []storedStamp, wantPlacements map[stampPair]stampPlacement) {
	t.Helper()

	for pair, placement := range wantPlacements {
		target, err := accountForRole(accounts, pair.target)
		if err != nil {
			t.Fatalf("役割%sのアカウントの取得に失敗: %v", pair.target, err)
		}

		positions := keptPostPositions(t, ctx, tx, target.profile.ID)

		var got []int
		for _, stamp := range stamps {
			if stamp.pair != pair {
				continue
			}

			// 画面から辿り着けないポストへのスタンプは、アプリケーションの
			// どの実行も生み出さない。削除されたポストはスタンプも一緒に持っていき、
			// 削除されたプロフィールのポストは表示されないためである。
			position, ok := positions[stamp.postID]
			if !ok {
				t.Errorf("スタンプ%s -> %sが、画面から辿り着けないポスト%sへ押されている",
					pair.source, pair.target, uuid.UUID(stamp.postID))
				continue
			}

			got = append(got, position)
		}

		want := make([]int, 0, placement.count)
		for i := range placement.count {
			want = append(want, placement.offset+i*stampStride)
		}

		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("スタンプ%s -> %sが押されたポストの位置 (新しい順) = %v、期待値 = %v",
				pair.source, pair.target, got, want)
			continue
		}

		assertStampsShareTheScreen(t, pair, got)
	}
}

// assertStampsShareTheScreenは、スタンプされた位置がポストカードの2つの
// 状態をどちらも見える状態に保つことを検証する。一覧を開いた画面にスタンプがあること、
// 2つのスタンプの間に未スタンプのポストがあること、スタンプの無い画面が1つも
// 無いことの3つになる。
//
// 境界はstampStrideから導かず、それ自身の数として書く。間隔を変える変更が、その
// 間隔の存在理由となっている性質を保つようにするためである。
func assertStampsShareTheScreen(t *testing.T, pair stampPair, positions []int) {
	t.Helper()

	if len(positions) == 0 {
		return
	}

	if positions[0] >= firstScreenPosts {
		t.Errorf("スタンプ%s -> %sの最初のスタンプの位置 = %d、期待値 = %d未満 (開いた画面にスタンプが1件も無い)",
			pair.source, pair.target, positions[0], firstScreenPosts)
	}

	for i := 1; i < len(positions); i++ {
		gap := positions[i] - positions[i-1]

		if gap < minStampGap {
			t.Errorf("スタンプ%s -> %sの位置%dと%dの間隔 = %d、期待値 = %d以上 (未スタンプのポストが間に入らない)",
				pair.source, pair.target, positions[i-1], positions[i], gap, minStampGap)
		}
		if gap > firstScreenPosts {
			t.Errorf("スタンプ%s -> %sの位置%dと%dの間隔 = %d、期待値 = %d以下 (スタンプの無い画面ができる)",
				pair.source, pair.target, positions[i-1], positions[i], gap, firstScreenPosts)
		}
	}
}

// keptPostPositionsは、プロフィールのポストそれぞれが、画面が一覧する順序の
// どこに位置するのかを、新しいものから0で数えて返す。画面から辿り着けないポストは
// 含めない。
func keptPostPositions(t *testing.T, ctx context.Context, tx *sql.Tx, profileID model.ProfileID) map[model.PostID]int {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT posts.id
		FROM posts
		JOIN profiles ON profiles.id = posts.profile_id
		WHERE posts.profile_id = $1
		  AND posts.discarded_at IS NULL
		  AND profiles.discarded_at IS NULL
		ORDER BY posts.published_at DESC, posts.id DESC
	`, uuid.UUID(profileID))
	if err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	positions := make(map[model.PostID]int)
	for rows.Next() {
		var id uuid.UUID

		if err := rows.Scan(&id); err != nil {
			t.Fatalf("ポストの読み取りに失敗: %v", err)
		}

		positions[model.PostID(id)] = len(positions)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}

	return positions
}

// assertStampInstantsは、すべてのスタンプが、すでに公開されたポストへ、
// すでに始まっている実行によって押されたことを検証する。
func assertStampInstants(t *testing.T, stamps []storedStamp, now time.Time) {
	t.Helper()

	for _, stamp := range stamps {
		if stamp.stampedAt.Before(stamp.publishedAt) {
			t.Errorf("スタンプ%s -> %sのstamped_at = %v、期待値 = 公開日時%v以降",
				stamp.pair.source, stamp.pair.target, stamp.stampedAt, stamp.publishedAt)
		}

		if stamp.stampedAt.After(storedInstant(now)) {
			t.Errorf("スタンプ%s -> %sのstamped_at = %v、期待値 = 実行の時点%v以前",
				stamp.pair.source, stamp.pair.target, stamp.stampedAt, storedInstant(now))
		}

		// 生成した履歴の作成日時と発生日時は同じ時点を表す。
		// 通知の表示順を決めるのはnotified_atとidである。
		if !stamp.createdAt.Equal(stamp.stampedAt) {
			t.Errorf("スタンプ%s -> %sのcreated_at = %v、期待値 = %v",
				stamp.pair.source, stamp.pair.target, stamp.createdAt, stamp.stampedAt)
		}
	}
}

// assertStampNotificationsは、すべてのスタンプが受け取り手へ通知を1つ渡した
// こと、そしてその通知が、それが何についてのものであるかのスタンプへ解決できることを
// 検証する。
func assertStampNotifications(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	accounts []seedAccount,
	stamps []storedStamp,
) {
	t.Helper()

	notifications := readStampNotifications(t, ctx, tx, accounts)

	if len(notifications) != len(stamps) {
		t.Errorf("通知の件数 = %d、期待値 = %d", len(notifications), len(stamps))
	}

	for _, stamp := range stamps {
		notification, ok := notifications[stamp.id]
		if !ok {
			t.Errorf("スタンプ%s -> %sに対応する通知が無い", stamp.pair.source, stamp.pair.target)
			continue
		}

		// 通知一覧は、通知が何についてのものかをこのカラムの組から解決する。
		// 一覧が知らない型は、表示できない行になる。
		if notification.notifiableType != "StampRecord" {
			t.Errorf("通知のnotifiable_type = %q、期待値 = %q", notification.notifiableType, "StampRecord")
		}

		// 通知は、スタンプを押した側ではなくポストの作者へ届く。アカウントが
		// 受け取った通知ではなく起こした通知を表示する一覧は、スタンプが双方向に
		// ある場所ではどこでも正しく見えることになる。
		if notification.pair != stamp.pair {
			t.Errorf("スタンプ%s -> %sの通知の向き = %s -> %s",
				stamp.pair.source, stamp.pair.target, notification.pair.source, notification.pair.target)
		}

		if !notification.notifiedAt.Equal(stamp.stampedAt) {
			t.Errorf("スタンプ%s -> %sの通知のnotified_at = %v、期待値 = %v",
				stamp.pair.source, stamp.pair.target, notification.notifiedAt, stamp.stampedAt)
		}

		if !notification.createdAt.Equal(stamp.stampedAt) {
			t.Errorf("スタンプ%s -> %sの通知のcreated_at = %v、期待値 = %v",
				stamp.pair.source, stamp.pair.target, notification.createdAt, stamp.stampedAt)
		}
	}
}

// readStampsは、データベースのスタンプを、それぞれが押されたポストと、その
// 両端の役割とともに返す。
func readStamps(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount) []storedStamp {
	t.Helper()

	roles := rolesByProfileID(t, accounts)

	rows, err := tx.QueryContext(ctx, `
		SELECT
			stamps.id, stamps.profile_id, stamps.post_id, stamps.stamped_at, stamps.created_at,
			posts.profile_id, posts.published_at
		FROM stamps
		JOIN posts ON posts.id = stamps.post_id
		ORDER BY stamps.stamped_at, stamps.id
	`)
	if err != nil {
		t.Fatalf("スタンプの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var stamps []storedStamp
	for rows.Next() {
		var stamp storedStamp
		var sourceID, postID, targetID uuid.UUID

		if err := rows.Scan(
			&stamp.id, &sourceID, &postID, &stamp.stampedAt, &stamp.createdAt,
			&targetID, &stamp.publishedAt,
		); err != nil {
			t.Fatalf("スタンプの読み取りに失敗: %v", err)
		}

		stamp.postID = model.PostID(postID)
		stamp.pair = stampPair{
			source: roles[model.ProfileID(sourceID)],
			target: roles[model.ProfileID(targetID)],
		}
		stamps = append(stamps, stamp)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("スタンプの取得に失敗: %v", err)
	}

	return stamps
}

// readStampNotificationsは、データベースの通知を、それぞれが何についての
// ものであるかのスタンプのIDごとに返す。
func readStampNotifications(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	accounts []seedAccount,
) map[uuid.UUID]storedNotification {
	t.Helper()

	roles := rolesByProfileID(t, accounts)

	rows, err := tx.QueryContext(ctx, `
		SELECT source_profile_id, target_profile_id, notifiable_type, notifiable_id, notified_at, created_at
		FROM notifications
	`)
	if err != nil {
		t.Fatalf("通知の取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	notifications := make(map[uuid.UUID]storedNotification)
	for rows.Next() {
		var notification storedNotification
		var sourceID, targetID uuid.UUID

		if err := rows.Scan(
			&sourceID, &targetID, &notification.notifiableType, &notification.notifiableID,
			&notification.notifiedAt, &notification.createdAt,
		); err != nil {
			t.Fatalf("通知の読み取りに失敗: %v", err)
		}

		notification.pair = stampPair{
			source: roles[model.ProfileID(sourceID)],
			target: roles[model.ProfileID(targetID)],
		}
		notifications[notification.notifiableID] = notification
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("通知の取得に失敗: %v", err)
	}

	return notifications
}
