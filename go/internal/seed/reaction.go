package seed

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
)

// stampNotifiableTypeは、スタンプについての通知が、それが何についての
// ものかを名指しするときの値。
//
// 値がRailsのクラス名であるのは、このカラムを書くのがdelegated_typeであるため。
// 通知は自身の対象を型とIDで指し、その型は、対象となるレコードクラスの名前になる。
// 別の値を書くシードは、通知一覧が解決できない行を作ることになる。
const stampNotifiableType = "StampRecord"

// stampStrideは、スタンプされたポストがプロフィールのポストの中でどれだけ
// 離れて置かれるか。新しいものからstampStride件に1件がスタンプされる。
//
// 最も新しいポストから続けて取らず、間隔を空ける。ポストカードは閲覧者がそれを
// スタンプしたかどうかを示すが、上から順にすべてがスタンプされている画面では、
// その2つの状態が一度に1つずつしか見えず、もう一方はその塊を過ぎるまで
// スクロールしないと現れない。
const stampStride = 2

// stampDelayは、公開からスタンプまでの最大待ち時間。
const stampDelay = 90 * time.Minute

// stampSpecは、ある役割が別の役割のポストをスタンプすること。その件数と、
// 相手のポストのどこからスタンプを始めるか。
type stampSpec struct {
	source seedRole
	target seedRole
	count  int

	// offsetは、最初のスタンプを相手のポストのどこへ置くか。最も新しいものから
	// 数える。
	//
	// 同じプロフィールをスタンプする2つの役割を、同じポストから引き離すためのもの。
	// ポストがいつスタンプされるのかは、それがいつ公開されたのかから求まり、通知一覧は
	// その時点で並べる。同じポストを選んだ2つの役割は、時点を共有する通知の組を
	// 受け取り手へ渡すことになる。それは、アプリケーションのどの実行も生み出さない
	// 一覧である。
	//
	// 値はstampStrideより小さい。stampStride以上のoffsetは、その範囲内のoffsetが
	// 既に覆っているポストへ着く。
	offset int
}

// stampSpecsは、誰が誰にスタンプを押すか。件数はamtから取る。
//
// roleMainとroleFollowerの間の2つの向きは件数が異なり、その双方をここに持つ。
// 通知一覧はこの組の両端から読まれるため、片方の向きしか持たない実行は、もう一方の
// 端のアカウントに空の一覧を残すことになる。それは、その画面を確認できない唯一の
// 状態である。
//
// roleNewcomerはroleMainのポストをスタンプするが、押し返されない。通知カードが
// フォローボタンを出すのは、閲覧者がフォローしていない相手のときだけであり、roleMainが
// フォローしていない役割はroleNewcomerだけである。この向きが無ければ、そのボタンは
// 実行が生み出すどの画面にも現れない。押し返しが無いのは、roleNewcomerが誰かに
// スタンプされる先を1件も書いていないためであり、そこに残る空の一覧こそ、サインアップ
// 直後の画面を読むためのものになる。
func stampSpecs(amt amounts) []stampSpec {
	return []stampSpec{
		{source: roleFollower, target: roleMain, count: amt.followerToMainStamps, offset: 0},
		{source: roleMain, target: roleFollower, count: amt.mainToFollowerStamps, offset: 0},
		{source: roleNewcomer, target: roleMain, count: amt.newcomerToMainStamps, offset: 1},
	}
}

// stampablePostは、スタンプを押せるポスト1件。どのポストであるかと、
// いつ公開されたか。
type stampablePost struct {
	id          model.PostID
	publishedAt time.Time
}

// reactionWriterは、実行1回分のスタンプと、それが起こす通知を書き込む。
type reactionWriter struct {
	tx  *sql.Tx
	now time.Time
}

// createReactionsは、各役割が別の役割のポストへ押したスタンプと、それを
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
			return fmt.Errorf("役割%sから役割%sへのスタンプの作成に失敗: %w", spec.source, spec.target, err)
		}
	}

	return nil
}

// writeStampsは、specが求めるスタンプをsourceからtargetのポストへ押す。
func (w *reactionWriter) writeStamps(ctx context.Context, source, target seedAccount, spec stampSpec) error {
	if spec.count <= 0 {
		return nil
	}

	posts, err := w.stampablePosts(ctx, target.profile.ID, spec.offset, spec.count)
	if err != nil {
		return err
	}

	// スタンプを押したい件数よりポストの少ないプロフィールは、押せるところまで
	// 押すのではなく報告する。amountsの件数はポストを生成する件数と読み合わせる
	// ものであり、黙って少ない件数を書き込む実行は、その食い違いをここではなく画面で
	// 気付かせることになる。
	if len(posts) < spec.count {
		return fmt.Errorf(
			"スタンプを押せるポストが%d件必要ですが、%d件しかありません",
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

// stampablePostsは、スタンプを押す先となるプロフィールのポストを最大count
// 件、古い順に返す。新しいものから数えてoffset番目のポストから始め、そこから遡って
// stampStride件に1件を取る。
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
	// ROW_NUMBERは1から、offsetは0から数える。そのため求めるポストは、
	// positionをstampStrideで割った余りが、offset+1をstampStrideで割った余りに
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

// writeStampは、ポストへスタンプを1つ押し、その作者へ通知を渡す。
//
// 2つの行を1文で書くのは、どちらか一方だけが書かれることが無いようにするため。
// スタンプは通知が何についてのものかであり、通知が自身の対象として名指しするもので
// ある。スタンプだけの行は受け取り手へ知らされないスタンプであり、通知だけの行は、
// 一覧が表示するものへ解決できない行になる。
//
// この書き込みをリポジトリのメソッドではなくここに置くのは、スタンプがRailsの
// 行う操作であるため。シードだけが呼ぶCreateは、Go版がほかに持たない経路を
// これらのテーブルへ向けて開けることになる。
func (w *reactionWriter) writeStamp(ctx context.Context, source, target seedAccount, post stampablePost) error {
	// 生成した履歴の作成日時と発生日時を揃える。
	// 通知一覧の表示順を決めるのはnotified_atとidである。
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

// stampedAtは、publishedAtに公開されたポストが、nowを基準とする実行で
// スタンプされる時刻を返す。
//
// スタンプは、公開からstampDelay後と、公開からnowまでの中間のうち
// 早い方へ置く。経過時間がstampDelayの2倍未満なら中間を使う。
// これにより生成したポストのスタンプ日時は公開日時に対して単調になり、nowより前に収まる。
func stampedAt(publishedAt, now time.Time) time.Time {
	delay := stampDelay
	if remaining := now.Sub(publishedAt) / 2; remaining < delay {
		delay = remaining
	}

	return storedInstant(publishedAt.Add(delay))
}
