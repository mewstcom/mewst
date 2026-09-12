package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

// stampNotifiableType is what a notification about a stamp names as the thing
// it is about.
//
// The value is the Rails class name, because the column is written by a
// delegated_type: a notification points at its subject with a type and an ID,
// and the type is the name of the record class that subject is. A seed that
// wrote anything else would produce rows the notification list cannot resolve.
//
// [Ja] stampNotifiableType は、スタンプについての通知が、それが何についての
// ものかを名指しするときの値。
//
// 値が Rails のクラス名であるのは、このカラムを書くのが delegated_type であるため。
// 通知は自身の対象を型とIDで指し、その型は、対象となるレコードクラスの名前になる。
// 別の値を書くシードは、通知一覧が解決できない行を作ることになる。
const stampNotifiableType = "StampRecord"

// stampStride is how far apart the stamped posts are placed in a profile's
// posts: one post in every stampStride of the newest ones is stamped.
//
// They are spaced rather than taken in a run from the newest post. A post card
// shows whether the viewer has stamped it, and a screen whose top rows are all
// stamped shows one of those two states at a time, with the other one only
// after scrolling past the whole block.
//
// [Ja] stampStride は、スタンプされたポストがプロフィールのポストの中でどれだけ
// 離れて置かれるか。新しいものから stampStride 件に 1 件がスタンプされる。
//
// 最も新しいポストから続けて取らず、間隔を空ける。ポストカードは閲覧者がそれを
// スタンプしたかどうかを示すが、上から順にすべてがスタンプされている画面では、
// その 2 つの状態が一度に 1 つずつしか見えず、もう一方はその塊を過ぎるまで
// スクロールしないと現れない。
const stampStride = 2

// stampDelay is the maximum delay between publication and a stamp.
//
// [Ja] stampDelay は、公開からスタンプまでの最大待ち時間。
const stampDelay = 90 * time.Minute

// stampSpec is one role stamping another's posts: how many of them, and where
// among the target's posts the stamps start.
//
// [Ja] stampSpec は、ある役割が別の役割のポストをスタンプすること。その件数と、
// 相手のポストのどこからスタンプを始めるか。
type stampSpec struct {
	source seedRole
	target seedRole
	count  int

	// offset is where the first stamp is placed among the target's posts,
	// counted from the newest one.
	//
	// It is what keeps two roles stamping the same profile off the same posts.
	// When a post is stamped is worked out from when it was published, and the
	// notification list orders by that instant, so two roles that chose the
	// same posts would hand the recipient pairs of notifications sharing an
	// instant, which is a list no run of the application produces.
	//
	// It is smaller than stampStride. An offset of stampStride or more lands
	// on the posts an offset within it already covers.
	//
	// [Ja] offset は、最初のスタンプを相手のポストのどこへ置くか。最も新しいものから
	// 数える。
	//
	// 同じプロフィールをスタンプする 2 つの役割を、同じポストから引き離すためのもの。
	// ポストがいつスタンプされるのかは、それがいつ公開されたのかから求まり、通知一覧は
	// その時点で並べる。同じポストを選んだ 2 つの役割は、時点を共有する通知の組を
	// 受け取り手へ渡すことになる。それは、アプリケーションのどの実行も生み出さない
	// 一覧である。
	//
	// 値は stampStride より小さい。stampStride 以上の offset は、その範囲内の offset が
	// 既に覆っているポストへ着く。
	offset int
}

// stampSpecs is who stamps whom, with the counts taken from amt.
//
// The two directions between roleMain and roleFollower are not the same size,
// and both of them are here. The notification list is read from both ends of
// that pair, so a run with one direction only would leave the account on the
// other end of it with an empty list, which is the one state that screen
// cannot be checked in.
//
// roleNewcomer stamps roleMain's posts without being stamped back. A
// notification card offers a follow button only for a profile the viewer has
// not followed, and roleNewcomer is the only role roleMain has not followed,
// so without this direction that button is on no screen a run produces.
// Nothing is stamped back because roleNewcomer has written nothing for anyone
// to stamp, and the empty list that leaves it with is the one the screens just
// after signing up are read from.
//
// [Ja] stampSpecs は、誰が誰にスタンプを押すか。件数は amt から取る。
//
// roleMain と roleFollower の間の 2 つの向きは件数が異なり、その双方をここに持つ。
// 通知一覧はこの組の両端から読まれるため、片方の向きしか持たない実行は、もう一方の
// 端のアカウントに空の一覧を残すことになる。それは、その画面を確認できない唯一の
// 状態である。
//
// roleNewcomer は roleMain のポストをスタンプするが、押し返されない。通知カードが
// フォローボタンを出すのは、閲覧者がフォローしていない相手のときだけであり、roleMain が
// フォローしていない役割は roleNewcomer だけである。この向きが無ければ、そのボタンは
// 実行が生み出すどの画面にも現れない。押し返しが無いのは、roleNewcomer が誰かに
// スタンプされる先を 1 件も書いていないためであり、そこに残る空の一覧こそ、サインアップ
// 直後の画面を読むためのものになる。
func stampSpecs(amt amounts) []stampSpec {
	return []stampSpec{
		{source: roleFollower, target: roleMain, count: amt.followerToMainStamps, offset: 0},
		{source: roleMain, target: roleFollower, count: amt.mainToFollowerStamps, offset: 0},
		{source: roleNewcomer, target: roleMain, count: amt.newcomerToMainStamps, offset: 1},
	}
}

// stampablePost is one post a stamp can be put on: which post it is, and when
// it was published.
//
// [Ja] stampablePost は、スタンプを押せるポスト 1 件。どのポストであるかと、
// いつ公開されたか。
type stampablePost struct {
	id          model.PostID
	publishedAt time.Time
}

// reactionWriter writes the stamps of one run, and the notifications they
// raise.
//
// [Ja] reactionWriter は、実行 1 回分のスタンプと、それが起こす通知を書き込む。
type reactionWriter struct {
	tx  *sql.Tx
	now time.Time
}

// createReactions writes the stamps each role put on another's posts, and the
// notifications the recipients were given.
//
// It runs after the posts because a stamp is put on one of them: which posts
// are stamped is chosen from what each profile has written, so a run that
// stamped before the posts existed would have nothing to choose from.
//
// [Ja] createReactions は、各役割が別の役割のポストへ押したスタンプと、それを
// 受け取った側へ与えられた通知を書き込む。
//
// ポストの後に実行する。スタンプはポストへ押すものであるため。どのポストが
// スタンプされるのかは各プロフィールが書いたものの中から選ばれるのであり、ポストが
// 存在しないうちにスタンプする実行には、選ぶ先が無い。
func createReactions(ctx context.Context, tx *sql.Tx, amt amounts, accounts []seedAccount, now time.Time) error {
	writer := &reactionWriter{tx: tx, now: now}

	for _, spec := range stampSpecs(amt) {
		source, err := accountForRole(accounts, spec.source)
		if err != nil {
			return err
		}

		target, err := accountForRole(accounts, spec.target)
		if err != nil {
			return err
		}

		if err := writer.writeStamps(ctx, source, target, spec); err != nil {
			return fmt.Errorf("役割 %s から役割 %s へのスタンプの作成に失敗: %w", spec.source, spec.target, err)
		}
	}

	return nil
}

// writeStamps puts the stamps spec asks for from source on target's posts.
//
// [Ja] writeStamps は、spec が求めるスタンプを source から target のポストへ押す。
func (w *reactionWriter) writeStamps(ctx context.Context, source, target seedAccount, spec stampSpec) error {
	if spec.count <= 0 {
		return nil
	}

	posts, err := w.stampablePosts(ctx, target.profile.ID, spec.offset, spec.count)
	if err != nil {
		return err
	}

	// A profile with fewer posts than the run wants stamped is reported rather
	// than stamped as far as it goes. The counts in amounts are read together
	// with the counts the posts are generated from, and a run that quietly
	// wrote fewer stamps would leave that disagreement to be noticed on a
	// screen instead of here.
	//
	// [Ja] スタンプを押したい件数よりポストの少ないプロフィールは、押せるところまで
	// 押すのではなく報告する。amounts の件数はポストを生成する件数と読み合わせる
	// ものであり、黙って少ない件数を書き込む実行は、その食い違いをここではなく画面で
	// 気付かせることになる。
	if len(posts) < spec.count {
		return fmt.Errorf(
			"スタンプを押せるポストが %d 件必要ですが、%d 件しかありません",
			spec.count, len(posts),
		)
	}

	for _, post := range posts {
		if err := w.writeStamp(ctx, source, target, post); err != nil {
			return err
		}
	}

	return nil
}

// stampablePosts returns up to count of a profile's posts to stamp, oldest
// first, starting at the offset-th newest post and taking one in every
// stampStride from there backwards.
//
// Only the posts a screen could reach are offered. A deleted post takes its
// stamps with it when it goes, and the posts of a deleted profile are not
// shown, so a stamp on either would be a row no run of the application
// produces.
//
// [Ja] stampablePosts は、スタンプを押す先となるプロフィールのポストを最大 count
// 件、古い順に返す。新しいものから数えて offset 番目のポストから始め、そこから遡って
// stampStride 件に 1 件を取る。
//
// 提示するのは画面から辿り着けるポストだけである。削除されたポストは、無くなるときに
// スタンプも一緒に持っていき、削除されたプロフィールのポストは表示されない。どちらへ
// 押したスタンプも、アプリケーションのどの実行も生み出さない行になる。
func (w *reactionWriter) stampablePosts(
	ctx context.Context,
	profileID model.ProfileID,
	offset int,
	count int,
) ([]stampablePost, error) {
	// ROW_NUMBER counts from one while offset counts from zero, so the posts
	// wanted are the ones whose position leaves offset+1 as its remainder
	// modulo stampStride.
	//
	// [Ja] ROW_NUMBER は 1 から、offset は 0 から数える。そのため求めるポストは、
	// position を stampStride で割った余りが、offset+1 を stampStride で割った余りに
	// 等しいものになる。
	rows, err := w.tx.QueryContext(ctx, `
		WITH reachable AS (
			SELECT
				posts.id,
				posts.published_at,
				ROW_NUMBER() OVER (ORDER BY posts.published_at DESC, posts.id DESC) AS position
			FROM posts
			JOIN profiles ON profiles.id = posts.profile_id
			WHERE posts.profile_id = $1
			  AND posts.discarded_at IS NULL
			  AND profiles.discarded_at IS NULL
		),
		chosen AS (
			SELECT id, published_at, position
			FROM reachable
			WHERE position % $2 = $3
			ORDER BY position
			LIMIT $4
		)
		SELECT id, published_at FROM chosen ORDER BY published_at, id
	`, uuid.UUID(profileID), stampStride, (offset+1)%stampStride, count)
	if err != nil {
		return nil, fmt.Errorf("スタンプを押すポストの取得に失敗: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var posts []stampablePost
	for rows.Next() {
		var id uuid.UUID
		var publishedAt time.Time

		if err := rows.Scan(&id, &publishedAt); err != nil {
			return nil, fmt.Errorf("スタンプを押すポストの読み取りに失敗: %w", err)
		}

		posts = append(posts, stampablePost{id: model.PostID(id), publishedAt: publishedAt})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("スタンプを押すポストの読み取りに失敗: %w", err)
	}

	return posts, nil
}

// writeStamp puts one stamp on a post and hands its author the notification.
//
// The two rows are written in one statement so that neither can be written
// without the other. A stamp is what the notification is about, and it is what
// the notification names as its subject, so a stamp on its own is a stamp the
// recipient is never told about, and a notification on its own is a row the
// list cannot resolve to anything to display.
//
// The writes are here rather than behind repository methods: stamping is
// something Rails does, and a Create only the seed calls would be a way into
// these tables the Go version does not otherwise have.
//
// [Ja] writeStamp は、ポストへスタンプを 1 つ押し、その作者へ通知を渡す。
//
// 2 つの行を 1 文で書くのは、どちらか一方だけが書かれることが無いようにするため。
// スタンプは通知が何についてのものかであり、通知が自身の対象として名指しするもので
// ある。スタンプだけの行は受け取り手へ知らされないスタンプであり、通知だけの行は、
// 一覧が表示するものへ解決できない行になる。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、スタンプが Rails の
// 行う操作であるため。シードだけが呼ぶ Create は、Go 版がほかに持たない経路を
// これらのテーブルへ向けて開けることになる。
func (w *reactionWriter) writeStamp(ctx context.Context, source, target seedAccount, post stampablePost) error {
	// Align creation and event timestamps in the generated history.
	// notified_at and id determine the notification list's order.
	//
	// [Ja] 生成した履歴の作成日時と発生日時を揃える。
	// 通知一覧の表示順を決めるのは notified_at と id である。
	at := stampedAt(post.publishedAt, w.now)

	if _, err := w.tx.ExecContext(ctx, `
		WITH stamp AS (
			INSERT INTO stamps (profile_id, post_id, stamped_at, created_at, updated_at)
			VALUES ($1, $2, $3, $3, $3)
			RETURNING id
		)
		INSERT INTO notifications (
			source_profile_id, target_profile_id, notifiable_type, notifiable_id,
			notified_at, created_at, updated_at
		)
		SELECT $1, $4, $5, stamp.id, $3, $3, $3 FROM stamp
	`,
		uuid.UUID(source.profile.ID),
		uuid.UUID(post.id),
		at,
		uuid.UUID(target.profile.ID),
		stampNotifiableType,
	); err != nil {
		return fmt.Errorf("スタンプの作成に失敗: %w", err)
	}

	return nil
}

// stampedAt returns when a post published at publishedAt is stamped, for a run
// anchored to now.
//
// A stamp lands at the earlier of stampDelay after publication and the
// midpoint between publication and now. The midpoint applies when less than
// twice stampDelay has elapsed. This keeps timestamps monotonic with
// publication times and before now for the generated posts.
//
// [Ja] stampedAt は、publishedAt に公開されたポストが、now を基準とする実行で
// スタンプされる時刻を返す。
//
// スタンプは、公開から stampDelay 後と、公開から now までの中間のうち
// 早い方へ置く。経過時間が stampDelay の 2 倍未満なら中間を使う。
// これにより生成したポストのスタンプ日時は公開日時に対して単調になり、now より前に収まる。
func stampedAt(publishedAt, now time.Time) time.Time {
	delay := stampDelay
	if remaining := now.Sub(publishedAt) / 2; remaining < delay {
		delay = remaining
	}

	return storedInstant(publishedAt.Add(delay))
}
