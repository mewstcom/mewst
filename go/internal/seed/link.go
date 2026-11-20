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

// linkPostAnchorは、リンクカード付きのポストを、プロフィールのポストの
// どこから広げるか。ポストを置く区間は、そこから数えたその件数分だけ遡った位置から、
// 実行自身の時点までになる。
//
// 区間を日数ではなく件数で測るのは、プロフィールの新しいポストが覆う時間の幅が
// 一定ではないため。ある月のポストは、その月のうち既に過ぎた部分へ敷かれるので、
// 月初に行われた実行はその月の取り分を数時間へ詰め込むことになり、固定の長さの
// 区間は、リンクカード付きのポストをその100件の下へ置くことになる。
const linkPostAnchor = 8

// seedLinkは、実行1回分が持つリンクカードの1件を、それを生み出したはずの
// 取得が残した形で表したもの。
//
// ドメインはここに持たない。それはページが自身について述べるものではなく、取得が
// canonical URLから導出するものであるため、シードも同じように導出する。URLの
// かたわらに手で書いたドメインは、それが名指しするはずのURLと食い違いうる2つ目の
// 場所になる。
type seedLink struct {
	canonicalURL string
	title        string

	// imageURLはOGP画像。リンクが持たないこともある。カードは画像がある
	// ときだけそれを描くため、その2つの体裁を見るには、画像を持つ1件と持たない
	// 1件の双方が要る。
	//
	// 画像は読み込まれない。それが属するページと同じくドキュメント用ドメインの下に
	// あるアドレスであるため、画像を持つカードが見せるのは、ドメインとタイトルの上に
	// 画像が確保する場所であって、絵ではない。
	imageURL string
}

// seedLinksは、実行1回分が作成するリンクカード。アドレスはexample.comの
// 下にある。ドキュメントのために取り分けられたドメインであるためで、実在のページを
// 指すリンクを持つシードは、カードを見た人を、誰も選んでいないサイトへ向けて開発環境の
// 外へ誘うことになる。
//
// ホストは互いに異なる。カードはタイトルの上にホストを表示するが、どのカードも
// example.comと読める実行では、その行がカードのものではなくリンクのものであることを
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
		// 折り返すのに都合のよい箇所を持たないタイトル。カードはタイトルへ
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

// linkPostは、リンクカードを持たせるために書くポスト1件。どのアカウントが
// 書くのか、何と書くのか、seedLinksのどれを引くのかを持つ。
//
// リンクをここに持たずcanonical URLで名指しするのは、リンクが、複数のポストが
// 引きうる独立した行であるため。同じURLを名指しする2件のポストは1つの行を
// 共有する。これは、既知のURLを渡されたときにアプリケーションが行うことである。
type linkPost struct {
	role         seedRole
	body         string
	canonicalURL string
}

// linkPostsは、カードを持たせるポスト。
//
// roleMainとroleFollowerへ置く。roleMainは開発者がサインインするアカウントで
// あるため、その本人のポストは、探しに行かずともカードに出会う場所になる。
// roleFollowerはroleMainにフォローされているため、そのポストは、カードが描かれる
// もう一方の画面であるホームタイムラインへカードを載せるものになる。
//
// 最後の1件は、最初の1件と同じリンクを引く。1つのリンク行が2件のポストを
// 持つ状態は、同じURLが2度ポストされるたびにアプリケーションが辿り着く状態で
// あり、これが無い実行では、ポストごとに1行を書くシードが、canonical URLの一意
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

// linkWriterは、実行1回分のリンクと、それを持つポストを書き込む。
type linkWriter struct {
	tx        *sql.Tx
	posts     *postWriter
	links     *repository.LinkRepository
	postLinks *repository.PostLinkRepository
}

// createLinksは、リンクカードと、それを引くポストを書き込む。
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

	// 役割は、linkPostsがたまたま名指しする順ではなく名簿自身の順で辿る。
	// どのアカウントを先に書くのかが、どの生成器についても1箇所で決まるようにする
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
			return fmt.Errorf("役割%sのリンクカード付きポストの作成に失敗: %w", role, err)
		}
	}

	return nil
}

// linkPostsForRoleは、roleがカードを持たせるポストを、linkPostsが名指しする
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

// writeLinksはseedLinksの1件につき1行を書き込み、canonical URLで
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
			return nil, fmt.Errorf("リンク%sの作成に失敗: %w", entry.canonicalURL, err)
		}

		links[entry.canonicalURL] = link
	}

	return links, nil
}

// writeLinkPostsは、あるアカウントのポストを書き込み、それぞれが引くカードを
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
			return fmt.Errorf("リンク%sがseedLinksにありません", entry.canonicalURL)
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

// anchorは、あるプロフィールのリンクカード付きポストが、そこから先へ広げられる
// 公開日時を返す。そのプロフィールが持つlinkPostAnchor番目に新しいポストのものか、
// それより少ない件数しか持たない場合は最も古いポストのものになる。
//
// 数えるのは画面に表示されるポストだけである。削除されたポストはどの画面にも無い
// ため、それを数えると、リンクカード付きのポストは、読み手には見えない1つ分だけ
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

	// ポストを1件も持たないプロフィールには、独自の区間を与えず報告する。
	// リンクカード付きのポストが置かれるのは、既にあるポストの中の位置であり、それを
	// 1件も持たないプロフィールは、実行がカードを与えるつもりでなかった役割である。
	if !anchored.Valid {
		return time.Time{}, fmt.Errorf("リンクカードを添えるポストの配置基準となるポストがありません")
	}

	return anchored.Time, nil
}

// linkDomainは、リンクのcanonical URLのホストを返す。カードがタイトルの上に
// 表示するのがそれである。
//
// 取り方はメタデータの取得と同じにする。シードが書いたリンクが、アプリケーションが
// 書きえたリンクであるようにするため。
func linkDomain(canonicalURL string) (string, error) {
	parsed, err := url.Parse(canonicalURL)
	if err != nil {
		return "", fmt.Errorf("リンクのURL %qの解析に失敗: %w", canonicalURL, err)
	}

	if parsed.Hostname() == "" {
		return "", fmt.Errorf("リンクのURL %qがホストを持っていません", canonicalURL)
	}

	return parsed.Hostname(), nil
}
