package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// backfilledTimelinePosts is how many of a profile's existing posts reach the
// home timeline of somebody who has just followed it.
//
// Following does two things to a timeline: the posts written from then on
// arrive one at a time as they are published, and the newest of the posts that
// were already there are copied in at once. The second is what keeps a
// timeline from being empty until the people in it write again, and it is
// capped, so a profile with three years of posts does not hand three years of
// rows to everyone who follows it.
//
// [Ja] backfilledTimelinePosts は、あるプロフィールをフォローしたとき、その時点で
// 既にあるポストのうち何件がフォローした側のホームタイムラインへ届くか。
//
// フォローはタイムラインに 2 つのことを行う。それ以降に書かれたポストは公開される
// たびに 1 件ずつ届き、その時点で既にあったポストのうち新しいものは一度にまとめて
// 写される。後者があることで、タイムラインはそこにいる人たちが次に書くまで空のまま
// にならない。そして後者には上限がある。3 年分のポストを持つプロフィールが、
// フォローした人すべてへ 3 年分の行を渡さないようにするためである。
const backfilledTimelinePosts = 30

// followEdge is one profile following another, and how long ago it did.
//
// [Ja] followEdge は、あるプロフィールが別のプロフィールをフォローしていること、
// そしてそれがどれだけ前のことなのか。
type followEdge struct {
	source seedRole
	target seedRole

	// monthsAgo is how far back followed_at is placed, counted from the
	// instant the run is anchored to. It decides what the follower's timeline
	// holds: everything the target published since, and the newest
	// backfilledTimelinePosts of what it had published before.
	//
	// [Ja] monthsAgo は、実行が基準とする時点から数えて、followed_at をどれだけ
	// 過去へ置くか。フォローした側のタイムラインが何を持つのかはこれが決める。
	// 相手がそれ以降に公開したもののすべてと、それより前に公開していたもののうち
	// 新しい backfilledTimelinePosts 件になる。
	monthsAgo int
}

// followEdges is the follow graph a run creates.
//
// roleMain and roleFollower follow each other, which is what gives both of
// them a home timeline with somebody else's posts in it. roleMain also follows
// roleEnglish without being followed back, so that the two counts a profile
// shows are different numbers: a screen that displayed the same count for both
// would look right anywhere the graph is symmetric.
//
// The dates are staggered rather than shared. roleFollower's is the older one,
// and it is far enough back that roleMain had already written more than
// backfilledTimelinePosts posts by then, so its timeline is one where the cap
// applied. roleMain followed back a month later, and reached roleEnglish only
// recently.
//
// [Ja] followEdges は、実行が作成するフォローの構成。
//
// roleMain と roleFollower は互いをフォローしており、それが両者のホームタイムラインを
// 他人のポストが並ぶ状態にする。roleMain は roleEnglish もフォローするが、そちらから
// フォローは返らない。プロフィールが示す 2 つの数を異なる数にするためである。両方に
// 同じ数を表示してしまう画面は、グラフが対称である場所ではどこでも正しく見える。
//
// 日付は揃えずにずらしている。roleFollower のものが古く、その時点で roleMain は
// 既に backfilledTimelinePosts 件を超えるポストを書いていた位置に置いてあるため、
// そのタイムラインは上限が効いた状態になる。roleMain はその 1 か月後にフォローを
// 返し、roleEnglish へは最近になって届いた。
var followEdges = []followEdge{
	{source: roleFollower, target: roleMain, monthsAgo: 6},
	{source: roleMain, target: roleFollower, monthsAgo: 5},
	{source: roleMain, target: roleEnglish, monthsAgo: 2},
}

// suggestedFollowSource is the role the suggested follows are offered to.
//
// It is roleNewcomer because that is the account that follows nobody: the
// screen that offers them is the one an account sees while it has no timeline
// of its own yet, and it is the account with no follows that can be looked at
// there.
//
// [Ja] suggestedFollowSource は、おすすめフォローが提示される役割。
//
// roleNewcomer であるのは、それが誰もフォローしていないアカウントであるため。
// おすすめを提示する画面は、まだ自分のタイムラインを持たないアカウントが見るもので
// あり、そこで見られるのはフォローを持たないアカウントになる。
const suggestedFollowSource = roleNewcomer

// suggestedFollowTargets are the profiles suggested to suggestedFollowSource.
//
// roleDiscarded is left out. The screen only shows suggestions whose profile
// is still there, so a suggestion pointing at a deleted profile would be a row
// that no screen displays.
//
// The order is the one the screen lists them in. It orders the suggested
// profiles by profiles.created_at, descending, which roleProfile's
// createdMinutesAgo staggers so that this list decides what a developer
// sees rather than the order the rows happen to come back in.
//
// [Ja] suggestedFollowTargets は、suggestedFollowSource へ提示されるプロフィール。
//
// roleDiscarded は含めない。画面が表示するのはプロフィールが残っているおすすめだけ
// であり、削除済みのプロフィールを指すおすすめは、どの画面にも表示されない行になる。
//
// 並びは画面が一覧する順序でもある。画面はおすすめのプロフィールを
// profiles.created_at の降順で並べる。roleProfile の createdMinutesAgo がこの
// カラムをずらしており、開発者が目にする順序を、行がたまたま返ってきた順序ではなく
// この一覧が決めるようにしている。
var suggestedFollowTargets = []seedRole{roleMain, roleFollower, roleEnglish}

// timelinePost is one post as a home timeline holds it: which post it is, and
// when it was published.
//
// The publication time is carried alongside rather than joined back to at read
// time, which is what lets a timeline be ordered without reaching for the
// posts themselves.
//
// [Ja] timelinePost は、ホームタイムラインが持つ形でのポスト 1 件。どのポストで
// あるかと、いつ公開されたか。
//
// 公開日時を読み出し時に join し直さず一緒に持つことで、タイムラインはポスト自体を
// 参照することなく並べ替えられる。
type timelinePost struct {
	id          model.PostID
	publishedAt time.Time
}

// followWriter writes the follows of one run, and the rows that follow from
// them.
//
// [Ja] followWriter は、実行 1 回分のフォローと、そこから生じる行を書き込む。
type followWriter struct {
	tx           *sql.Tx
	homeTimeline *repository.HomeTimelinePostRepository
	now          time.Time
}

// createFollows writes the home timelines, the follow graph, and the suggested
// follows offered to the account that has none.
//
// A home timeline is filled from two sides. Each profile's own posts are on it
// because writing a post puts it there, and the posts of whoever it follows
// are on it because the follow handed them over.
//
// It runs after the posts because a timeline is made of them: a follow written
// before its target had written anything would leave the follower with an
// empty timeline and no way to tell that apart from a fanout that failed.
//
// [Ja] createFollows は、ホームタイムラインと、フォローの構成、そしてフォローを
// 持たないアカウントへ提示されるおすすめフォローを書き込む。
//
// ホームタイムラインは 2 つの側から埋まる。プロフィール自身のポストは、ポストを
// 書くことがそれをそこへ置くために載り、フォローしている相手のポストは、フォローが
// それを渡したために載る。
//
// ポストの後に実行する。タイムラインはポストからできているためである。相手がまだ
// 何も書いていないうちに書かれたフォローは、フォローした側に空のタイムラインを残し、
// それを配信の失敗と区別する手立てを残さない。
func createFollows(ctx context.Context, tx *sql.Tx, accounts []seedAccount, now time.Time) error {
	writer := &followWriter{
		tx:           tx,
		homeTimeline: repository.NewHomeTimelinePostRepository(query.New(tx)),
		now:          now,
	}

	if err := writer.writeOwnTimelines(ctx, accounts); err != nil {
		return err
	}

	for _, edge := range followEdges {
		source, err := accountForRole(accounts, edge.source)
		if err != nil {
			return err
		}

		target, err := accountForRole(accounts, edge.target)
		if err != nil {
			return err
		}

		if err := writer.writeFollow(ctx, source, target, edge.monthsAgo); err != nil {
			return fmt.Errorf("役割 %s から役割 %s へのフォローの作成に失敗: %w", edge.source, edge.target, err)
		}
	}

	if err := writer.writeSuggestedFollows(ctx, accounts); err != nil {
		return err
	}

	return nil
}

// writeFollow records the follow and fills the follower's timeline with what
// the follow gave it.
//
// The write is here rather than behind a repository method: following is
// something Rails does, and a Create only the seed calls would be a way into
// the graph the Go version does not otherwise have.
//
// [Ja] writeFollow は、フォローを記録し、そのフォローが与えたものでフォローした側の
// タイムラインを埋める。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、フォローが Rails の
// 行う操作であるため。シードだけが呼ぶ Create は、Go 版がほかに持たない経路を
// フォローの構成へ向けて開けることになる。
func (w *followWriter) writeFollow(ctx context.Context, source, target seedAccount, monthsAgo int) error {
	// created_at is the follow's own instant rather than the run's. A follow
	// is created at the moment it is made, and the screens that order follows
	// read that column, so a graph whose rows were all created at once would
	// be listed in an order the followed_at dates contradict.
	//
	// [Ja] created_at には実行の時点ではなくフォロー自身の時点を入れる。フォローは
	// それが行われた瞬間に作成されるものであり、フォローを並べる画面はこのカラムを
	// 読む。すべての行が同時に作成されたグラフは、followed_at の日付が否定する順序で
	// 並べられることになる。
	followedAt := storedInstant(w.now.AddDate(0, -monthsAgo, 0))

	if _, err := w.tx.ExecContext(ctx, `
		INSERT INTO follows (source_profile_id, target_profile_id, followed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $3, $3)
	`, uuid.UUID(source.profile.ID), uuid.UUID(target.profile.ID), followedAt); err != nil {
		return fmt.Errorf("フォローの作成に失敗: %w", err)
	}

	posts, err := w.timelinePosts(ctx, target.profile.ID, followedAt)
	if err != nil {
		return err
	}

	return w.fillTimeline(ctx, source.profile.ID, posts)
}

// writeOwnTimelines puts the posts each profile wrote on its own home timeline.
//
// The application does this as part of creating a post, beside the delivery to
// the followers, so that what was just written is on its writer's own home
// screen without waiting for the fanout. A run that filled timelines from the
// follow graph alone would give every account a home screen holding everybody's
// posts but its own, which is a state no run of the application produces.
//
// [Ja] writeOwnTimelines は、それぞれのプロフィールが書いたポストを、そのプロフィール
// 自身のホームタイムラインへ置く。
//
// アプリケーションはこれを、フォロワーへの配信とあわせて、ポストの作成の一部として
// 行う。たった今書かれたものが、配信を待たずに書いた本人のホーム画面にあるように
// するためである。フォローの構成だけからタイムラインを埋める実行は、どのアカウント
// にも、自分以外のすべての人のポストを持つホーム画面を与えることになり、それは
// アプリケーションのどの実行も生み出さない状態である。
func (w *followWriter) writeOwnTimelines(ctx context.Context, accounts []seedAccount) error {
	for _, account := range accounts {
		posts, err := w.ownTimelinePosts(ctx, account.profile.ID)
		if err != nil {
			return fmt.Errorf("役割 %s の自身のポストの取得に失敗: %w", account.roster.role, err)
		}

		if err := w.fillTimeline(ctx, account.profile.ID, posts); err != nil {
			return fmt.Errorf("役割 %s の自身のホームタイムラインの作成に失敗: %w", account.roster.role, err)
		}
	}

	return nil
}

// ownTimelinePosts returns the posts a profile put on its own home timeline,
// oldest first.
//
// The two conditions are the ones timelinePosts applies, so that both sides of
// home_timeline_posts agree on which posts a row can be written for. A deleted
// post has its timeline rows removed with it, and a deleted profile is not one
// anybody signs in as, so its own home screen is one no run opens.
//
// [Ja] ownTimelinePosts は、あるプロフィールが自身のホームタイムラインへ置いた
// ポストを、古いものから順に返す。
//
// 2 つの条件は timelinePosts が適用するものと同じにする。home_timeline_posts の
// 双方の側が、どのポストに対して行を書けるのかについて一致しているようにするため。
// 削除されたポストはタイムラインの行も一緒に取り除かれ、削除済みプロフィールは
// 誰かがサインインするためのものではないため、その自身のホーム画面を開く実行は無い。
func (w *followWriter) ownTimelinePosts(ctx context.Context, profileID model.ProfileID) ([]timelinePost, error) {
	rows, err := w.tx.QueryContext(ctx, `
		SELECT posts.id, posts.published_at
		FROM posts
		JOIN profiles ON profiles.id = posts.profile_id
		WHERE posts.profile_id = $1
		  AND posts.discarded_at IS NULL
		  AND profiles.discarded_at IS NULL
		ORDER BY posts.published_at, posts.id
	`, uuid.UUID(profileID))
	if err != nil {
		return nil, fmt.Errorf("自身のタイムラインへ入れるポストの取得に失敗: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var posts []timelinePost
	for rows.Next() {
		var id uuid.UUID
		var publishedAt time.Time

		if err := rows.Scan(&id, &publishedAt); err != nil {
			return nil, fmt.Errorf("自身のタイムラインへ入れるポストの読み取りに失敗: %w", err)
		}

		posts = append(posts, timelinePost{id: model.PostID(id), publishedAt: publishedAt})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("自身のタイムラインへ入れるポストの読み取りに失敗: %w", err)
	}

	return posts, nil
}

// fillTimeline puts posts on the home timeline of profileID.
//
// [Ja] fillTimeline は、profileID のホームタイムラインへ posts を置く。
func (w *followWriter) fillTimeline(ctx context.Context, profileID model.ProfileID, posts []timelinePost) error {
	for _, post := range posts {
		if _, err := w.homeTimeline.Create(ctx, repository.CreateHomeTimelinePostInput{
			ProfileID:   profileID,
			PostID:      post.id,
			PublishedAt: post.publishedAt,
		}); err != nil {
			return fmt.Errorf("ホームタイムラインへの追加に失敗: %w", err)
		}
	}

	return nil
}

// timelinePosts returns what a follow made at followedAt put into the
// follower's home timeline, oldest first.
//
// The two halves of the query are the two ways a post reaches a timeline. The
// first is delivery: every post published since the follow was made is handed
// to the follower as it is published. The second is the backfill: the newest
// of the posts that were already there are copied in when the follow is made,
// up to backfilledTimelinePosts of them.
//
// Only the posts a screen could reach are taken. A deleted post has its
// timeline rows removed with it, and the posts of a deleted profile are not
// shown, so a timeline holding either would be one no run of the application
// produces.
//
// [Ja] timelinePosts は、followedAt に行われたフォローがフォローした側のホーム
// タイムラインへ入れたものを、古い順に返す。
//
// クエリの 2 つの部分は、ポストがタイムラインへ届く 2 通りの経路にあたる。1 つ目は
// 配信で、フォロー以降に公開されたポストは、公開されるたびにフォローした側へ渡される。
// 2 つ目は遡っての取り込みで、その時点で既にあったポストのうち新しいものが、フォローの
// 際にまとめて写される。件数は backfilledTimelinePosts までとなる。
//
// 取るのは画面から辿り着けるポストだけである。削除されたポストはタイムラインの行も
// 一緒に取り除かれ、削除されたプロフィールのポストは表示されない。どちらかを持つ
// タイムラインは、アプリケーションのどの実行も生み出さないものになる。
func (w *followWriter) timelinePosts(
	ctx context.Context,
	profileID model.ProfileID,
	followedAt time.Time,
) ([]timelinePost, error) {
	rows, err := w.tx.QueryContext(ctx, `
		WITH reachable AS (
			SELECT posts.id, posts.published_at
			FROM posts
			JOIN profiles ON profiles.id = posts.profile_id
			WHERE posts.profile_id = $1
			  AND posts.discarded_at IS NULL
			  AND profiles.discarded_at IS NULL
		)
		SELECT id, published_at FROM reachable WHERE published_at >= $2
		UNION ALL
		(
			SELECT id, published_at FROM reachable WHERE published_at < $2
			ORDER BY published_at DESC, id DESC
			LIMIT $3
		)
		ORDER BY published_at, id
	`, uuid.UUID(profileID), followedAt, backfilledTimelinePosts)
	if err != nil {
		return nil, fmt.Errorf("タイムラインへ入れるポストの取得に失敗: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var posts []timelinePost
	for rows.Next() {
		var id uuid.UUID
		var publishedAt time.Time

		if err := rows.Scan(&id, &publishedAt); err != nil {
			return nil, fmt.Errorf("タイムラインへ入れるポストの読み取りに失敗: %w", err)
		}

		posts = append(posts, timelinePost{id: model.PostID(id), publishedAt: publishedAt})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("タイムラインへ入れるポストの読み取りに失敗: %w", err)
	}

	return posts, nil
}

// writeSuggestedFollows offers suggestedFollowTargets to
// suggestedFollowSource.
//
// The suggestions are written rather than derived. The application builds them
// from the profiles the followees of an account follow, which an account that
// follows nobody has none of, so deriving them would leave the one account
// that is there to see the screen with nothing to see on it.
//
// [Ja] writeSuggestedFollows は、suggestedFollowSource へ
// suggestedFollowTargets を提示する。
//
// おすすめは導出せずに書き込む。アプリケーションはこれを、アカウントがフォローして
// いる相手のフォロー先から組み立てる。誰もフォローしていないアカウントはそれを 1 つも
// 持たないため、導出に任せると、その画面を見るためにいる唯一のアカウントに、見るものが
// 何も無い状態を残すことになる。
func (w *followWriter) writeSuggestedFollows(ctx context.Context, accounts []seedAccount) error {
	source, err := accountForRole(accounts, suggestedFollowSource)
	if err != nil {
		return err
	}

	for _, role := range suggestedFollowTargets {
		target, err := accountForRole(accounts, role)
		if err != nil {
			return err
		}

		// checked_at is left unset. It is what the screen marks a suggestion
		// with once it has been looked at, and a suggestion that has been
		// looked at is one the screen stops offering.
		//
		// [Ja] checked_at は未設定のままにする。これは、おすすめが一度見られたことを
		// 画面が記すためのものであり、見られたおすすめは、画面が提示をやめるものである。
		if _, err := w.tx.ExecContext(ctx, `
			INSERT INTO suggested_follows (source_profile_id, target_profile_id, created_at, updated_at)
			VALUES ($1, $2, $3, $3)
		`, uuid.UUID(source.profile.ID), uuid.UUID(target.profile.ID), storedInstant(w.now)); err != nil {
			return fmt.Errorf("役割 %s へのおすすめフォローの作成に失敗: %w", role, err)
		}
	}

	return nil
}
