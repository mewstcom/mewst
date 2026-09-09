package seed

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
)

// linkPostAnchor is how far down a profile's posts the link posts are spread
// from: the window they are placed in reaches from that many posts back up to
// the run's own instant.
//
// The window is measured in posts rather than in days because how much time a
// profile's newest posts cover is not fixed. A month's posts are laid across
// the part of the month that has already happened, so a run on the first of a
// month packs that month's share into a few hours, and a window of a fixed
// length would put the link posts below a hundred of them.
//
// [Ja] linkPostAnchor は、リンクカード付きのポストを、プロフィールのポストの
// どこから広げるか。ポストを置く区間は、そこから数えたその件数分だけ遡った位置から、
// 実行自身の時点までになる。
//
// 区間を日数ではなく件数で測るのは、プロフィールの新しいポストが覆う時間の幅が
// 一定ではないため。ある月のポストは、その月のうち既に過ぎた部分へ敷かれるので、
// 月初に行われた実行はその月の取り分を数時間へ詰め込むことになり、固定の長さの
// 区間は、リンクカード付きのポストをその 100 件の下へ置くことになる。
const linkPostAnchor = 8

// seedLink is one link card a run holds, as the fetch that would have produced
// it left it.
//
// The domain is not here. It is what the fetch derives from the canonical URL
// rather than something a page says about itself, so the seed derives it the
// same way: a domain written by hand beside the URL is a second place for it
// to disagree with the URL it is supposed to name.
//
// [Ja] seedLink は、実行 1 回分が持つリンクカードの 1 件を、それを生み出したはずの
// 取得が残した形で表したもの。
//
// ドメインはここに持たない。それはページが自身について述べるものではなく、取得が
// canonical URL から導出するものであるため、シードも同じように導出する。URL の
// かたわらに手で書いたドメインは、それが名指しするはずの URL と食い違いうる 2 つ目の
// 場所になる。
type seedLink struct {
	canonicalURL string
	title        string

	// imageURL is the OGP image, which a link may not have. The card draws the
	// image only when there is one, so both an entry with an image and an
	// entry without are needed for that pair of layouts to be seen.
	//
	// The images do not load. They are addressed under the documentation
	// domain like the pages they belong to, so what a card with one shows is
	// the space the image reserves above the domain and the title, not a
	// picture.
	//
	// [Ja] imageURL は OGP 画像。リンクが持たないこともある。カードは画像がある
	// ときだけそれを描くため、その 2 つの体裁を見るには、画像を持つ 1 件と持たない
	// 1 件の双方が要る。
	//
	// 画像は読み込まれない。それが属するページと同じくドキュメント用ドメインの下に
	// あるアドレスであるため、画像を持つカードが見せるのは、ドメインとタイトルの上に
	// 画像が確保する場所であって、絵ではない。
	imageURL string
}

// seedLinks are the link cards a run creates. Their addresses are under
// example.com, which is the domain set aside for documentation: a seed whose
// links pointed at real pages would be inviting whoever looks at a card to
// leave the development environment for a site nobody chose.
//
// The hosts differ from one another. The card shows the host above the title,
// and a run whose every card read example.com could not show that the line
// belongs to the link rather than to the card.
//
// [Ja] seedLinks は、実行 1 回分が作成するリンクカード。アドレスは example.com の
// 下にある。ドキュメントのために取り分けられたドメインであるためで、実在のページを
// 指すリンクを持つシードは、カードを見た人を、誰も選んでいないサイトへ向けて開発環境の
// 外へ誘うことになる。
//
// ホストは互いに異なる。カードはタイトルの上にホストを表示するが、どのカードも
// example.com と読める実行では、その行がカードのものではなくリンクのものであることを
// 示せない。
var seedLinks = []seedLink{
	{
		canonicalURL: "https://example.com/articles/hand-drip-basics",
		title:        "ハンドドリップの基本 - 湯温と蒸らしの時間",
		imageURL:     "https://example.com/images/hand-drip-basics.png",
	},
	{
		canonicalURL: "https://blog.example.com/riverside-walking-course",
		title:        "川沿いを歩く - 朝の 30 分でまわれるコース",
	},
	{
		// A title with nowhere convenient to break. The card gives the title
		// the width it has and no more, so a long one is where a card either
		// wraps or pushes the post it sits in sideways.
		//
		// [Ja] 折り返すのに都合のよい箇所を持たないタイトル。カードはタイトルへ
		// 持っている幅だけを与えるため、長いタイトルは、カードが折り返すか、それが
		// 収まっているポストを横へ押し広げるかの分かれ目になる。
		canonicalURL: "https://docs.example.com/photography/exposure/compensation-on-cloudy-days",
		title:        "曇りの日の露出補正について、測光モードの選び方から仕上がりの色の傾向まで一通りまとめたページ",
	},
	{
		canonicalURL: "https://books.example.com/bakery-guide",
		title:        "町のパン屋を巡る本",
		imageURL:     "https://books.example.com/images/bakery-guide.png",
	},
}

// linkPost is one post written to carry a link card: which account writes it,
// what it says, and which of seedLinks it cites.
//
// The link is named by its canonical URL rather than held here, because a link
// is a row of its own that several posts can cite. Two posts naming the same
// URL share one row, which is what the application does when it is handed a
// URL it already knows.
//
// [Ja] linkPost は、リンクカードを持たせるために書くポスト 1 件。どのアカウントが
// 書くのか、何と書くのか、seedLinks のどれを引くのかを持つ。
//
// リンクをここに持たず canonical URL で名指しするのは、リンクが、複数のポストが
// 引きうる独立した行であるため。同じ URL を名指しする 2 件のポストは 1 つの行を
// 共有する。これは、既知の URL を渡されたときにアプリケーションが行うことである。
type linkPost struct {
	role         seedRole
	body         string
	canonicalURL string
}

// linkPosts are the posts that carry the cards.
//
// They go to roleMain and roleFollower. roleMain is the account a developer
// signs in as, so its own posts are where a card is met without going looking
// for one; roleFollower is followed by roleMain, so its posts are what puts a
// card on a home timeline, which is the other screen the card is drawn on.
//
// The last of them cites the link the first one does. One link row carrying
// two posts is the state the application reaches whenever a URL is posted a
// second time, and a run without it would let a seed that wrote one row per
// post pass unnoticed until the unique index over the canonical URL refused it.
//
// [Ja] linkPosts は、カードを持たせるポスト。
//
// roleMain と roleFollower へ置く。roleMain は開発者がサインインするアカウントで
// あるため、その本人のポストは、探しに行かずともカードに出会う場所になる。
// roleFollower は roleMain にフォローされているため、そのポストは、カードが描かれる
// もう一方の画面であるホームタイムラインへカードを載せるものになる。
//
// 最後の 1 件は、最初の 1 件と同じリンクを引く。1 つのリンク行が 2 件のポストを
// 持つ状態は、同じ URL が 2 度ポストされるたびにアプリケーションが辿り着く状態で
// あり、これが無い実行では、ポストごとに 1 行を書くシードが、canonical URL の一意
// インデックスに拒まれるまで気付かれずに済んでしまう。
var linkPosts = []linkPost{
	{
		role:         roleMain,
		body:         "この記事のとおりに淹れてみた。蒸らしを長めに取ると、確かに味が変わる。",
		canonicalURL: "https://example.com/articles/hand-drip-basics",
	},
	{
		role:         roleMain,
		body:         "散歩コースを探していて見つけたページ。次の休みに歩いてみる。",
		canonicalURL: "https://blog.example.com/riverside-walking-course",
	},
	{
		role:         roleMain,
		body:         "曇りの日の写真がうまく撮れないので読んでいる。",
		canonicalURL: "https://docs.example.com/photography/exposure/compensation-on-cloudy-days",
	},
	{
		role:         roleFollower,
		body:         "近所のパン屋がこの本に載っていた。",
		canonicalURL: "https://books.example.com/bakery-guide",
	},
	{
		role:         roleFollower,
		body:         "流れてきた記事。パンを焼くときの湯温にも同じことが言えそう。",
		canonicalURL: "https://example.com/articles/hand-drip-basics",
	},
}

// linkWriter writes the links of one run, and the posts that carry them.
//
// [Ja] linkWriter は、実行 1 回分のリンクと、それを持つポストを書き込む。
type linkWriter struct {
	tx        *sql.Tx
	posts     *postWriter
	links     *repository.LinkRepository
	postLinks *repository.PostLinkRepository
}

// createLinks writes the link cards and the posts that cite them.
//
// It runs after the everyday posts and before the follows. Where a link post
// goes is worked out from the posts a profile already holds, which a run that
// wrote it first would have none of; and a home timeline is filled from the
// posts that exist when the follow is written, so a link post written after
// the follows would be on no timeline but the profile it was written under.
//
// [Ja] createLinks は、リンクカードと、それを引くポストを書き込む。
//
// 日常ポストの後、フォローの前に実行する。リンクカード付きのポストをどこへ置くのかは、
// プロフィールが既に持っているポストから求まるものであり、先に書き込む実行にはそれが
// 無い。またホームタイムラインは、フォローが書かれた時点で存在するポストから埋められる
// ため、フォローの後に書かれたリンクカード付きのポストは、それが書かれたプロフィール
// 以外のどのタイムラインにも載らない。
func createLinks(
	ctx context.Context,
	tx *sql.Tx,
	applicationID model.OauthApplicationID,
	accounts []seedAccount,
	now time.Time,
) error {
	writer := &linkWriter{
		tx:        tx,
		posts:     newPostWriter(tx, applicationID, now),
		links:     repository.NewLinkRepository(query.New(tx)),
		postLinks: repository.NewPostLinkRepository(query.New(tx)),
	}

	links, err := writer.writeLinks(ctx)
	if err != nil {
		return err
	}

	// The roles are walked in the roster's own order rather than in the order
	// linkPosts happens to name them, so that which account is written first
	// is decided in one place for every generator.
	//
	// [Ja] 役割は、linkPosts がたまたま名指しする順ではなく名簿自身の順で辿る。
	// どのアカウントを先に書くのかが、どの生成器についても 1 箇所で決まるようにする
	// ため。
	for _, role := range allSeedRoles {
		posts := linkPostsForRole(role)
		if len(posts) == 0 {
			continue
		}

		account, err := accountForRole(accounts, role)
		if err != nil {
			return err
		}

		if err := writer.writeLinkPosts(ctx, account, posts, links); err != nil {
			return fmt.Errorf("役割 %s のリンクカード付きポストの作成に失敗: %w", role, err)
		}
	}

	return nil
}

// linkPostsForRole returns the posts role carries a card on, in the order
// linkPosts names them.
//
// [Ja] linkPostsForRole は、role がカードを持たせるポストを、linkPosts が名指しする
// 順に返す。
func linkPostsForRole(role seedRole) []linkPost {
	var posts []linkPost
	for _, post := range linkPosts {
		if post.role == role {
			posts = append(posts, post)
		}
	}

	return posts
}

// writeLinks writes one row per entry in seedLinks and returns them by
// canonical URL, which is what the posts cite them by.
//
// [Ja] writeLinks は seedLinks の 1 件につき 1 行を書き込み、canonical URL で
// 引ける形で返す。ポストがリンクを引くときに使うのがそれであるため。
func (w *linkWriter) writeLinks(ctx context.Context) (map[string]*model.Link, error) {
	links := make(map[string]*model.Link, len(seedLinks))

	for _, entry := range seedLinks {
		domain, err := linkDomain(entry.canonicalURL)
		if err != nil {
			return nil, err
		}

		link, err := w.links.Create(ctx, repository.CreateLinkInput{
			CanonicalURL: entry.canonicalURL,
			Domain:       domain,
			Title:        entry.title,
			ImageURL:     entry.imageURL,
		})
		if err != nil {
			return nil, fmt.Errorf("リンク %s の作成に失敗: %w", entry.canonicalURL, err)
		}

		links[entry.canonicalURL] = link
	}

	return links, nil
}

// writeLinkPosts writes the posts of one account and attaches the card each of
// them cites.
//
// [Ja] writeLinkPosts は、あるアカウントのポストを書き込み、それぞれが引くカードを
// 紐づける。
func (w *linkWriter) writeLinkPosts(
	ctx context.Context,
	account seedAccount,
	posts []linkPost,
	links map[string]*model.Link,
) error {
	start, err := w.anchor(ctx, account.profile.ID)
	if err != nil {
		return err
	}

	for index, entry := range posts {
		link, ok := links[entry.canonicalURL]
		if !ok {
			return fmt.Errorf("リンク %s が seedLinks にありません", entry.canonicalURL)
		}

		publishedAt := storedInstant(postTimeInWindow(start, w.posts.now, index, len(posts)))

		postID, err := w.posts.writePost(ctx, account, entry.body, publishedAt)
		if err != nil {
			return err
		}

		if _, err := w.postLinks.Create(ctx, repository.CreatePostLinkInput{
			PostID: postID,
			LinkID: link.ID,
		}); err != nil {
			return fmt.Errorf("リンクカードの紐付けに失敗: %w", err)
		}

		if err := w.posts.recordLastPostAt(ctx, account, publishedAt); err != nil {
			return err
		}
	}

	return nil
}

// anchor returns the publication time the link posts of a profile are spread
// forward from: that of the linkPostAnchor-th newest post it holds, or of its
// oldest one when it holds fewer than that.
//
// Only the posts a screen shows are counted. A deleted post is on no screen,
// so counting it would push the link posts down the list by a place a reader
// cannot see.
//
// [Ja] anchor は、あるプロフィールのリンクカード付きポストが、そこから先へ広げられる
// 公開日時を返す。そのプロフィールが持つ linkPostAnchor 番目に新しいポストのものか、
// それより少ない件数しか持たない場合は最も古いポストのものになる。
//
// 数えるのは画面に表示されるポストだけである。削除されたポストはどの画面にも無い
// ため、それを数えると、リンクカード付きのポストは、読み手には見えない 1 つ分だけ
// 一覧の下へ押し下げられることになる。
func (w *linkWriter) anchor(ctx context.Context, profileID model.ProfileID) (time.Time, error) {
	var anchored sql.NullTime

	if err := w.tx.QueryRowContext(ctx, `
		SELECT MIN(published_at) FROM (
			SELECT published_at
			FROM posts
			WHERE posts.profile_id = $1
			  AND posts.discarded_at IS NULL
			ORDER BY published_at DESC, id DESC
			LIMIT $2
		) newest
	`, uuid.UUID(profileID), linkPostAnchor).Scan(&anchored); err != nil {
		return time.Time{}, fmt.Errorf("リンクカード付きポストの配置基準の取得に失敗: %w", err)
	}

	// A profile with no posts is reported rather than given a window of its
	// own. Where a link post goes is a position among the posts that are
	// already there, and a profile that holds none is a role the run was not
	// meant to give a card to.
	//
	// [Ja] ポストを 1 件も持たないプロフィールには、独自の区間を与えず報告する。
	// リンクカード付きのポストが置かれるのは、既にあるポストの中の位置であり、それを
	// 1 件も持たないプロフィールは、実行がカードを与えるつもりでなかった役割である。
	if !anchored.Valid {
		return time.Time{}, fmt.Errorf("リンクカードを添えるポストの配置基準となるポストがありません")
	}

	return anchored.Time, nil
}

// linkDomain returns the host of a link's canonical URL, which is what the
// card shows above the title.
//
// It is taken the way the metadata fetch takes it, so that a link the seed
// wrote is one the application could have written.
//
// [Ja] linkDomain は、リンクの canonical URL のホストを返す。カードがタイトルの上に
// 表示するのがそれである。
//
// 取り方はメタデータの取得と同じにする。シードが書いたリンクが、アプリケーションが
// 書きえたリンクであるようにするため。
func linkDomain(canonicalURL string) (string, error) {
	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		return "", fmt.Errorf("リンクの URL %q の解析に失敗: %w", canonicalURL, err)
	}

	if parsed.Hostname() == "" {
		return "", fmt.Errorf("リンクの URL %q がホストを持っていません", canonicalURL)
	}

	return parsed.Hostname(), nil
}
