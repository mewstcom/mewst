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

// backfilledTimelinePostsは、あるプロフィールをフォローしたとき、その時点で
// 既にあるポストのうち何件がフォローした側のホームタイムラインへ届くか。
//
// フォローはタイムラインに2つのことを行う。それ以降に書かれたポストは公開される
// たびに1件ずつ届き、その時点で既にあったポストのうち新しいものは一度にまとめて
// 写される。後者があることで、タイムラインはそこにいる人たちが次に書くまで空のまま
// にならない。そして後者には上限がある。3年分のポストを持つプロフィールが、
// フォローした人すべてへ3年分の行を渡さないようにするためである。
const backfilledTimelinePosts = 30

// followEdgeは、あるプロフィールが別のプロフィールをフォローしていること、
// そしてそれがどれだけ前のことなのか。
type followEdge struct {
	source seedRole
	target seedRole

	// monthsAgoは、実行が基準とする時点から数えて、followed_atをどれだけ
	// 過去へ置くか。フォローした側のタイムラインが何を持つのかはこれが決める。
	// 相手がそれ以降に公開したもののすべてと、それより前に公開していたもののうち
	// 新しいbackfilledTimelinePosts件になる。
	monthsAgo int
}

// followEdgesは、実行が作成するフォローの構成。
//
// roleMainとroleFollowerは互いをフォローしており、それが両者のホームタイムラインを
// 他人のポストが並ぶ状態にする。roleMainはroleEnglishもフォローするが、そちらから
// フォローは返らない。プロフィールが示す2つの数を異なる数にするためである。両方に
// 同じ数を表示してしまう画面は、グラフが対称である場所ではどこでも正しく見える。
//
// 日付は揃えずにずらしている。roleFollowerのものが古く、その時点でroleMainは
// 既にbackfilledTimelinePosts件を超えるポストを書いていた位置に置いてあるため、
// そのタイムラインは上限が効いた状態になる。roleMainはその1か月後にフォローを
// 返し、roleEnglishへは最近になって届いた。
var followEdges = []followEdge{
	{source: roleFollower, target: roleMain, monthsAgo: 6},
	{source: roleMain, target: roleFollower, monthsAgo: 5},
	{source: roleMain, target: roleEnglish, monthsAgo: 2},
}

// suggestedFollowSourceは、おすすめフォローが提示される役割。
//
// roleNewcomerであるのは、それが誰もフォローしていないアカウントであるため。
// おすすめを提示する画面は、まだ自分のタイムラインを持たないアカウントが見るもので
// あり、そこで見られるのはフォローを持たないアカウントになる。
const suggestedFollowSource = roleNewcomer

// suggestedFollowTargetsは、suggestedFollowSourceへ提示されるプロフィール。
//
// roleDiscardedは含めない。画面が表示するのはプロフィールが残っているおすすめだけ
// であり、削除済みのプロフィールを指すおすすめは、どの画面にも表示されない行になる。
//
// 並びは画面が一覧する順序でもある。画面はおすすめのプロフィールを
// profiles.created_atの降順で並べる。roleProfileのcreatedMinutesAgoがこの
// カラムをずらしており、開発者が目にする順序を、行がたまたま返ってきた順序ではなく
// この一覧が決めるようにしている。
var suggestedFollowTargets = []seedRole{roleMain, roleFollower, roleEnglish}

// timelinePostは、ホームタイムラインが持つ形でのポスト1件。どのポストで
// あるかと、いつ公開されたか。
//
// 公開日時を読み出し時にjoinし直さず一緒に持つことで、タイムラインはポスト自体を
// 参照することなく並べ替えられる。
type timelinePost struct {
	id          model.PostID
	publishedAt time.Time
}

// followWriterは、実行1回分のフォローと、そこから生じる行を書き込む。
type followWriter struct {
	tx           *sql.Tx
	homeTimeline *repository.HomeTimelinePostRepository
	now          time.Time
}

// createFollowsは、ホームタイムラインと、フォローの構成、そしてフォローを
// 持たないアカウントへ提示されるおすすめフォローを書き込む。
//
// ホームタイムラインは2つの側から埋まる。プロフィール自身のポストは、ポストを
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
			return fmt.Errorf("役割%sから役割%sへのフォローの作成に失敗: %w", edge.source, edge.target, err)
		}
	}

	if err := writer.writeSuggestedFollows(ctx, accounts); err != nil {
		return err
	}

	return nil
}

// writeFollowは、フォローを記録し、そのフォローが与えたものでフォローした側の
// タイムラインを埋める。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、フォローがRailsの
// 行う操作であるため。シードだけが呼ぶCreateは、Go版がほかに持たない経路を
// フォローの構成へ向けて開けることになる。
func (w *followWriter) writeFollow(ctx context.Context, source, target seedAccount, monthsAgo int) error {
	// created_atには実行の時点ではなくフォロー自身の時点を入れる。フォローは
	// それが行われた瞬間に作成されるものであり、フォローを並べる画面はこのカラムを
	// 読む。すべての行が同時に作成されたグラフは、followed_atの日付が否定する順序で
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

// writeOwnTimelinesは、それぞれのプロフィールが書いたポストを、そのプロフィール
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
			return fmt.Errorf("役割%sの自身のポストの取得に失敗: %w", account.roster.role, err)
		}

		if err := w.fillTimeline(ctx, account.profile.ID, posts); err != nil {
			return fmt.Errorf("役割%sの自身のホームタイムラインの作成に失敗: %w", account.roster.role, err)
		}
	}

	return nil
}

// ownTimelinePostsは、あるプロフィールが自身のホームタイムラインへ置いた
// ポストを、古いものから順に返す。
//
// 2つの条件はtimelinePostsが適用するものと同じにする。home_timeline_postsの
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

// fillTimelineは、profileIDのホームタイムラインへpostsを置く。
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

// timelinePostsは、followedAtに行われたフォローがフォローした側のホーム
// タイムラインへ入れたものを、古い順に返す。
//
// クエリの2つの部分は、ポストがタイムラインへ届く2通りの経路にあたる。1つ目は
// 配信で、フォロー以降に公開されたポストは、公開されるたびにフォローした側へ渡される。
// 2つ目は遡っての取り込みで、その時点で既にあったポストのうち新しいものが、フォローの
// 際にまとめて写される。件数はbackfilledTimelinePostsまでとなる。
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

// writeSuggestedFollowsは、suggestedFollowSourceへ
// suggestedFollowTargetsを提示する。
//
// おすすめは導出せずに書き込む。アプリケーションはこれを、アカウントがフォローして
// いる相手のフォロー先から組み立てる。誰もフォローしていないアカウントはそれを1つも
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

		// checked_atは未設定のままにする。これは、おすすめが一度見られたことを
		// 画面が記すためのものであり、見られたおすすめは、画面が提示をやめるものである。
		if _, err := w.tx.ExecContext(ctx, `
			INSERT INTO suggested_follows (source_profile_id, target_profile_id, created_at, updated_at)
			VALUES ($1, $2, $3, $3)
		`, uuid.UUID(source.profile.ID), uuid.UUID(target.profile.ID), storedInstant(w.now)); err != nil {
			return fmt.Errorf("役割%sへのおすすめフォローの作成に失敗: %w", role, err)
		}
	}

	return nil
}
