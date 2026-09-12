package seed

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/validator"
)

// documentationDomain is the domain set aside for documentation, and the one
// every link of a run is addressed under.
//
// [Ja] documentationDomain は、ドキュメントのために取り分けられたドメイン。実行が
// 持つすべてのリンクは、この下のアドレスを指す。
const documentationDomain = "example.com"

// storedLinkPost is one card-carrying post read back out of the database,
// alongside the link it carries.
//
// [Ja] storedLinkPost は、データベースから読み戻したカード付きのポスト 1 件と、
// それが持つリンク。
type storedLinkPost struct {
	postID       model.PostID
	profileID    model.ProfileID
	content      string
	publishedAt  time.Time
	canonicalURL string
}

// storedLink is one link row read back out of the database.
//
// [Ja] storedLink は、データベースから読み戻したリンクの行 1 件。
type storedLink struct {
	domain   string
	title    string
	imageURL string
}

// TestSeedLinks verifies that every card a run holds is one the application
// could have fetched, and that the set of them covers the two layouts a card
// is drawn in.
//
// [Ja] TestSeedLinks は、実行が持つそれぞれのカードが、アプリケーションが取得
// しえたものであること、そしてそれらの組み合わせが、カードの描かれる 2 つの体裁を
// 覆っていることを検証する。
func TestSeedLinks(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool, len(seedLinks))
	var domains []string
	withImage, withoutImage := 0, 0

	for _, entry := range seedLinks {
		// The canonical URL has a unique index, so a duplicate entry is a run
		// that refuses to write its second link rather than one that writes
		// two rows.
		//
		// [Ja] canonical URL には一意インデックスがあるため、重複した 1 件は、
		// 2 行を書き込む実行ではなく、2 つ目のリンクの書き込みを拒まれる実行になる。
		if seen[entry.canonicalURL] {
			t.Errorf("リンク %s が seedLinks に重複している", entry.canonicalURL)
		}
		seen[entry.canonicalURL] = true

		if !validator.IsValidURL(entry.canonicalURL) {
			t.Errorf("リンクの canonical URL %q がアプリケーションの受け付ける形式でない", entry.canonicalURL)
		}

		if entry.title == "" {
			t.Errorf("リンク %s がタイトルを持っていない", entry.canonicalURL)
		}

		domain, err := linkDomain(entry.canonicalURL)
		if err != nil {
			t.Errorf("リンク %s のドメインの導出に失敗: %v", entry.canonicalURL, err)
			continue
		}
		assertDocumentationDomain(t, entry.canonicalURL, domain)
		domains = append(domains, domain)

		if entry.imageURL == "" {
			withoutImage++
			continue
		}

		withImage++
		if !validator.IsValidURL(entry.imageURL) {
			t.Errorf("リンク %s の画像 URL %q がアプリケーションの受け付ける形式でない",
				entry.canonicalURL, entry.imageURL)
		}

		imageDomain, err := linkDomain(entry.imageURL)
		if err != nil {
			t.Errorf("リンク %s の画像 URL のドメインの導出に失敗: %v", entry.canonicalURL, err)
			continue
		}
		assertDocumentationDomain(t, entry.imageURL, imageDomain)
	}

	// The card draws an image only when the link has one, so a run whose links
	// all had an image, or none did, would leave one of those two layouts on
	// no screen.
	//
	// [Ja] カードが画像を描くのは、リンクが画像を持つときだけである。すべてのリンクが
	// 画像を持つ実行も、どれも持たない実行も、その 2 つの体裁の一方をどの画面にも
	// 残さないことになる。
	if withImage == 0 {
		t.Error("画像を持つリンクが 1 件も無い")
	}
	if withoutImage == 0 {
		t.Error("画像を持たないリンクが 1 件も無い")
	}

	// The card shows the host above the title. A run whose every card read the
	// same host could not show that the line belongs to the link.
	//
	// [Ja] カードはタイトルの上にホストを表示する。どのカードも同じホストと読める
	// 実行では、その行がリンクのものであることを示せない。
	slices.Sort(domains)
	if len(slices.Compact(domains)) < 2 {
		t.Errorf("リンクのドメインが %v の 1 種類しかない", domains)
	}
}

// TestLinkPosts verifies that the posts carrying the cards cite links the run
// writes, are written by accounts whose posts a screen displays, and land
// where a card is met without scrolling for it.
//
// [Ja] TestLinkPosts は、カードを持つポストが、実行の書き込むリンクを引くこと、
// ポストが画面に表示されるアカウントによって書かれること、そしてスクロールせずに
// カードに出会える位置へ置かれることを検証する。
func TestLinkPosts(t *testing.T) {
	t.Parallel()

	catalog := make(map[string]bool, len(seedLinks))
	for _, entry := range seedLinks {
		catalog[entry.canonicalURL] = true
	}

	citations := make(map[string]int, len(seedLinks))
	perRole := make(map[seedRole]int, len(allSeedRoles))

	for _, post := range linkPosts {
		if !catalog[post.canonicalURL] {
			t.Errorf("リンクカード付きポストが seedLinks に無いリンク %s を引いている", post.canonicalURL)
		}
		citations[post.canonicalURL]++
		perRole[post.role]++

		if !slices.Contains(allSeedRoles, post.role) {
			t.Errorf("リンクカード付きポストに名簿の持たない役割 %s が含まれる", post.role)
		}

		// roleNewcomer holds no post at all, and the posts of roleDiscarded's
		// deleted profile are shown on no screen. A card given to either would
		// be one nobody can be shown.
		//
		// [Ja] roleNewcomer はポストを 1 件も持たず、roleDiscarded の削除済み
		// プロフィールのポストはどの画面にも表示されない。どちらかへ与えたカードは、
		// 誰にも見せられないカードになる。
		if post.role == roleNewcomer || post.role == roleDiscarded {
			t.Errorf("リンクカード付きポストが役割 %s へ与えられている", post.role)
		}

		if strings.TrimSpace(post.body) == "" {
			t.Errorf("リンク %s を引くポストの本文が空", post.canonicalURL)
		}
		if length := utf8.RuneCountInString(post.body); length > model.MaximumPostContentLength {
			t.Errorf("リンク %s を引くポストの本文の長さ = %d, want %d 以下",
				post.canonicalURL, length, model.MaximumPostContentLength)
		}
	}

	// A link no post cites is a row that reaches no card.
	//
	// [Ja] どのポストも引かないリンクは、どのカードにも届かない行になる。
	for _, entry := range seedLinks {
		if citations[entry.canonicalURL] == 0 {
			t.Errorf("リンク %s を引くポストが 1 件も無い", entry.canonicalURL)
		}
	}

	// One link carrying two posts is the state a URL posted a second time
	// reaches. Without it, a seed that wrote one row per post would go
	// unnoticed until the unique index refused it.
	//
	// [Ja] 1 つのリンクが 2 件のポストを持つ状態は、同じ URL が 2 度ポストされた
	// ときに辿り着く状態である。これが無ければ、ポストごとに 1 行を書くシードは、
	// 一意インデックスに拒まれるまで気付かれずに済んでしまう。
	reused := false
	for _, count := range citations {
		if count > 1 {
			reused = true
		}
	}
	if !reused {
		t.Error("2 件のポストが引くリンクが 1 つも無い")
	}

	// A card is drawn on the profile of whoever wrote the post and on the home
	// timeline of whoever follows them, so both of those screens need a role
	// that carries one.
	//
	// [Ja] カードは、ポストを書いた本人のプロフィールと、その人をフォローしている
	// 人のホームタイムラインに描かれる。そのどちらの画面にも、カードを持つ役割が
	// 要る。
	if perRole[roleMain] == 0 {
		t.Errorf("役割 %s がリンクカード付きポストを 1 件も持たない", roleMain)
	}

	onTimeline := false
	for _, edge := range followEdges {
		if edge.source == roleMain && perRole[edge.target] > 0 {
			onTimeline = true
		}
	}
	if !onTimeline {
		t.Errorf("役割 %s がフォローしている役割が、リンクカード付きポストを 1 件も持たない", roleMain)
	}

	// The window a link post is placed in starts at the linkPostAnchor-th
	// newest post, so the posts newer than one of them are at most the
	// linkPostAnchor-1 that came before the window and the other link posts of
	// the same account.
	//
	// [Ja] リンクカード付きのポストが置かれる区間は、linkPostAnchor 番目に新しい
	// ポストから始まる。そのため、あるリンクカード付きのポストより新しいポストは、
	// 区間より前にある linkPostAnchor-1 件と、同じアカウントの他のリンクカード付きの
	// ポストまでになる。
	most := 0
	for _, count := range perRole {
		most = max(most, count)
	}
	if bound := linkPostAnchor - 1 + most; bound > firstScreenPosts {
		t.Errorf("リンクカード付きポストが届きうる最も下の位置 = %d, want %d 以下", bound, firstScreenPosts)
	}
}

// TestLinkDomain verifies that a link's domain is the host of its URL, and
// that a URL without one is reported rather than written as a card with a
// blank line where the host goes.
//
// [Ja] TestLinkDomain は、リンクのドメインがその URL のホストであること、そして
// ホストを持たない URL が、ホストの入る行が空のカードとして書き込まれるのではなく
// 報告されることを検証する。
func TestLinkDomain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		url        string
		want       string
		wantErrMsg string
	}{
		{name: "ホストとパス", url: "https://example.com/articles/1", want: "example.com"},
		{name: "サブドメイン", url: "https://blog.example.com/1", want: "blog.example.com"},
		{name: "ポート付き", url: "https://example.com:8443/1", want: "example.com"},
		{name: "ホストなし", url: "/articles/1", wantErrMsg: "ホストを持っていません"},
		{name: "空文字列", url: "", wantErrMsg: "ホストを持っていません"},
		{name: "パース不能", url: "https://exa mple.com/1", wantErrMsg: "解析に失敗"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := linkDomain(test.url)

			if test.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("linkDomain(%q) がエラーを返さなかった", test.url)
				}
				if !strings.Contains(err.Error(), test.wantErrMsg) {
					t.Errorf("エラー = %q, want %q を含む", err, test.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("linkDomain(%q) に失敗: %v", test.url, err)
			}
			if got != test.want {
				t.Errorf("linkDomain(%q) = %q, want %q", test.url, got, test.want)
			}
		})
	}
}

// TestCreateLinks verifies that a run writes one row per link, attaches each
// of them to the posts that cite it, and places those posts where a card is
// met on the screen a profile and a timeline open on.
//
// [Ja] TestCreateLinks は、実行がリンクごとに 1 行を書き、それぞれをそれを引く
// ポストへ紐づけること、そしてそれらのポストを、プロフィールとタイムラインを開いた
// 画面でカードに出会える位置へ置くことを検証する。
func TestCreateLinks(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	// The instant is truncated to what the column can hold, so that a
	// published_at read back out of the database can be compared with the one
	// that was written rather than with a value rounded away from it.
	//
	// [Ja] 時点はカラムが保持できる精度へ切り詰める。データベースから読み戻した
	// published_at を、そこから丸められた値ではなく、書き込んだ値と比較できるように
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

	if err := createLinks(ctx, tx, applicationID, accounts, now); err != nil {
		t.Fatalf("リンクの作成に失敗: %v", err)
	}

	assertStoredLinks(t, ctx, tx, accounts)

	posts := readLinkPosts(t, ctx, tx, accounts)
	assertLinkPostsMatchRoster(t, posts, accounts)
	assertLinkPostPlacement(t, ctx, tx, accounts, posts, now)
}

// TestCreateLinks_ProfileWithoutPosts verifies that a profile holding no posts
// is reported rather than given a card, because where a card-carrying post
// goes is a position among the posts that are already there.
//
// [Ja] TestCreateLinks_ProfileWithoutPosts は、ポストを 1 件も持たないプロフィールが、
// カードを与えられるのではなく報告されることを検証する。カード付きのポストが置かれる
// のは、既にあるポストの中の位置であるため。
func TestCreateLinks_ProfileWithoutPosts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()
	now := time.Now().Truncate(time.Microsecond)

	accounts, err := createAccounts(ctx, tx, newTestRoster(t), now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	account, err := accountForRole(accounts, roleNewcomer)
	if err != nil {
		t.Fatalf("役割 %s のアカウントの取得に失敗: %v", roleNewcomer, err)
	}

	writer := &linkWriter{tx: tx}
	if _, err := writer.anchor(ctx, account.profile.ID); err == nil {
		t.Fatal("ポストを持たないプロフィールでエラーを返さなかった")
	} else if !strings.Contains(err.Error(), "配置基準となるポストがありません") {
		t.Errorf("エラー = %q, want %q を含む", err, "配置基準となるポストがありません")
	}
}

// assertDocumentationDomain verifies that a URL is addressed under the domain
// set aside for documentation, so that a card never sends whoever opens it to
// a site nobody chose.
//
// [Ja] assertDocumentationDomain は、URL がドキュメントのために取り分けられた
// ドメインの下にあることを検証する。カードが、それを開いた人を誰も選んでいない
// サイトへ送ることが決して無いようにするため。
func assertDocumentationDomain(t *testing.T, rawURL, domain string) {
	t.Helper()

	if domain != documentationDomain && !strings.HasSuffix(domain, "."+documentationDomain) {
		t.Errorf("URL %q のドメイン = %q, want %q かそのサブドメイン", rawURL, domain, documentationDomain)
	}
}

// assertStoredLinks compares the link rows with the catalog they were written
// from, including the domain, which the run derives rather than copies.
//
// The links are reached through the posts of the run's accounts. The test
// database is shared between packages, so a query that named the links by
// their canonical URL alone would answer with a row another test committed
// under the same URL, and compare this run's catalog against it.
//
// [Ja] assertStoredLinks は、リンクの行を、それが書き込まれた元の一覧と突き合わせる。
// 実行が書き写すのではなく導出するドメインも対象に含める。
//
// リンクは、実行のアカウントが持つポストを辿って引く。テスト用データベースは
// パッケージ間で共有されるため、canonical URL だけでリンクを名指しする問い合わせは、
// 他のテストが同じ URL でコミットした行とともに答え、この実行の一覧をそれと
// 突き合わせることになる。
func assertStoredLinks(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount) {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT links.canonical_url, links.domain, links.title, links.image_url
		FROM links
		JOIN post_links ON post_links.link_id = links.id
		JOIN posts ON posts.id = post_links.post_id
		WHERE posts.profile_id = ANY($1::uuid[])
	`, pq.Array(profileUUIDs(accounts)))
	if err != nil {
		t.Fatalf("リンクの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	stored := make(map[string]storedLink, len(seedLinks))
	for rows.Next() {
		var canonicalURL string
		var link storedLink

		if err := rows.Scan(&canonicalURL, &link.domain, &link.title, &link.imageURL); err != nil {
			t.Fatalf("リンクの読み取りに失敗: %v", err)
		}

		stored[canonicalURL] = link
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("リンクの取得に失敗: %v", err)
	}

	if len(stored) != len(seedLinks) {
		t.Errorf("リンクの件数 = %d, want %d", len(stored), len(seedLinks))
	}

	for _, entry := range seedLinks {
		link, ok := stored[entry.canonicalURL]
		if !ok {
			t.Errorf("リンク %s が作成されていない", entry.canonicalURL)
			continue
		}

		domain, err := linkDomain(entry.canonicalURL)
		if err != nil {
			t.Errorf("リンク %s のドメインの導出に失敗: %v", entry.canonicalURL, err)
			continue
		}

		if link.domain != domain {
			t.Errorf("リンク %s のドメイン = %q, want %q", entry.canonicalURL, link.domain, domain)
		}
		if link.title != entry.title {
			t.Errorf("リンク %s のタイトル = %q, want %q", entry.canonicalURL, link.title, entry.title)
		}
		if link.imageURL != entry.imageURL {
			t.Errorf("リンク %s の画像 URL = %q, want %q", entry.canonicalURL, link.imageURL, entry.imageURL)
		}
	}
}

// readLinkPosts returns the card-carrying posts of the run's accounts, oldest
// first.
//
// The rows are scoped to those accounts. The test database is shared between
// packages, so a query over the whole table would answer with the links other
// tests left behind.
//
// [Ja] readLinkPosts は、実行のアカウントが持つカード付きのポストを、古い順に返す。
//
// 行はそのアカウントに絞る。テスト用データベースはパッケージ間で共有されるため、
// テーブル全体への問い合わせは、他のテストが残したリンクとともに答えることになる。
func readLinkPosts(t *testing.T, ctx context.Context, tx *sql.Tx, accounts []seedAccount) []storedLinkPost {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT posts.id, posts.profile_id, posts.content, posts.published_at, links.canonical_url
		FROM post_links
		JOIN posts ON posts.id = post_links.post_id
		JOIN links ON links.id = post_links.link_id
		WHERE posts.profile_id = ANY($1::uuid[])
		ORDER BY posts.published_at, posts.id
	`, pq.Array(profileUUIDs(accounts)))
	if err != nil {
		t.Fatalf("リンクカード付きポストの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var posts []storedLinkPost
	for rows.Next() {
		var postID, profileID uuid.UUID
		var post storedLinkPost

		if err := rows.Scan(&postID, &profileID, &post.content, &post.publishedAt, &post.canonicalURL); err != nil {
			t.Fatalf("リンクカード付きポストの読み取りに失敗: %v", err)
		}

		post.postID = model.PostID(postID)
		post.profileID = model.ProfileID(profileID)
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("リンクカード付きポストの取得に失敗: %v", err)
	}

	return posts
}

// assertLinkPostsMatchRoster verifies that each role wrote the posts it was
// given, in the order they are written down, each carrying the link it cites.
//
// [Ja] assertLinkPostsMatchRoster は、それぞれの役割が、与えられたポストを書かれて
// いる順に書き、そのそれぞれが引くリンクを持つことを検証する。
func assertLinkPostsMatchRoster(t *testing.T, posts []storedLinkPost, accounts []seedAccount) {
	t.Helper()

	if len(posts) != len(linkPosts) {
		t.Errorf("リンクカード付きポストの件数 = %d, want %d", len(posts), len(linkPosts))
	}

	byProfile := make(map[model.ProfileID][]storedLinkPost, len(accounts))
	for _, post := range posts {
		byProfile[post.profileID] = append(byProfile[post.profileID], post)
	}

	for _, role := range allSeedRoles {
		want := linkPostsForRole(role)

		account, err := accountForRole(accounts, role)
		if err != nil {
			t.Fatalf("役割 %s のアカウントの取得に失敗: %v", role, err)
		}

		got := byProfile[account.profile.ID]
		if len(got) != len(want) {
			t.Errorf("役割 %s のリンクカード付きポストの件数 = %d, want %d", role, len(got), len(want))
			continue
		}

		for i, entry := range want {
			if got[i].content != entry.body {
				t.Errorf("役割 %s の %d 件目のポストの本文 = %q, want %q", role, i+1, got[i].content, entry.body)
			}
			if got[i].canonicalURL != entry.canonicalURL {
				t.Errorf("役割 %s の %d 件目のポストが引くリンク = %q, want %q",
					role, i+1, got[i].canonicalURL, entry.canonicalURL)
			}
		}
	}
}

// assertLinkPostPlacement verifies that every card-carrying post is on the
// screen its author's posts open on, was published before the run that wrote
// it, and is accounted for by the profile's newest post.
//
// [Ja] assertLinkPostPlacement は、すべてのカード付きのポストが、その作者のポストを
// 開いた画面の中にあること、それを書き込んだ実行より前に公開されていること、そして
// プロフィールの最も新しいポストがそれを織り込んでいることを検証する。
func assertLinkPostPlacement(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	accounts []seedAccount,
	posts []storedLinkPost,
	now time.Time,
) {
	t.Helper()

	positionsByProfile := make(map[model.ProfileID]map[model.PostID]int, len(accounts))

	for _, post := range posts {
		positions, ok := positionsByProfile[post.profileID]
		if !ok {
			positions = keptPostPositions(t, ctx, tx, post.profileID)
			positionsByProfile[post.profileID] = positions
		}

		// A card is worth writing where somebody meets it. A post that has to
		// be scrolled to is one a developer who opened the screen to look at
		// the card would not find on it.
		//
		// [Ja] カードは、人がそれに出会う場所にあってこそ書く意味を持つ。スクロール
		// しなければ辿り着けないポストは、カードを見るために画面を開いた開発者が、
		// そこに見つけられないポストである。
		position, ok := positions[post.postID]
		if !ok {
			t.Errorf("リンク %s を引くポストが、画面から辿り着けない", post.canonicalURL)
			continue
		}
		if position >= firstScreenPosts {
			t.Errorf("リンク %s を引くポストの位置 (新しい順) = %d, want %d 未満",
				post.canonicalURL, position, firstScreenPosts)
		}

		if !post.publishedAt.Before(storedInstant(now)) {
			t.Errorf("リンク %s を引くポストの公開日時 = %v, want 実行の時点 %v より前",
				post.canonicalURL, post.publishedAt, now)
		}

		account, err := accountForProfile(accounts, post.profileID)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if account.profile.LastPostAt == nil {
			t.Errorf("役割 %s の last_post_at が未設定", account.roster.role)
			continue
		}
		if account.profile.LastPostAt.Before(post.publishedAt) {
			t.Errorf("役割 %s の last_post_at = %v, want リンクカード付きポストの公開日時 %v 以降",
				account.roster.role, *account.profile.LastPostAt, post.publishedAt)
		}
	}
}

// profileUUIDs returns the profile IDs of the accounts in the form a query
// parameter takes them, which is what scopes a query to the rows this run
// wrote.
//
// [Ja] profileUUIDs は、アカウントのプロフィールIDを、クエリのパラメータが受け取る
// 形で返す。問い合わせをこの実行が書いた行へ絞るのに使うのがそれである。
func profileUUIDs(accounts []seedAccount) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, uuid.UUID(account.profile.ID))
	}

	return ids
}

// accountForProfile returns the account the profile belongs to, so that a post
// read back out of the database can be reported by the role that wrote it.
//
// [Ja] accountForProfile は、そのプロフィールが属するアカウントを返す。データベース
// から読み戻したポストを、それを書いた役割で報告できるようにするため。
func accountForProfile(accounts []seedAccount, profileID model.ProfileID) (seedAccount, error) {
	for _, account := range accounts {
		if account.profile.ID == profileID {
			return account, nil
		}
	}

	return seedAccount{}, fmt.Errorf("プロフィール %s のアカウントが見つかりません", uuid.UUID(profileID))
}
