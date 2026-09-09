package seed

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

// discardedVariationCount is how many of the variation posts are deleted after
// they are written. It is counted from the list rather than written down, so
// that a variation added to the list does not leave a stale number behind.
//
// [Ja] discardedVariationCount は、エクスポート確認用ポストのうち、書き込んだ
// あとに削除するものの件数。書き下さず一覧から数えるのは、一覧へ 1 件足したときに
// 古い数字が取り残されないようにするため。
func discardedVariationCount() int {
	count := 0
	for _, variation := range postVariations {
		if variation.discarded {
			count++
		}
	}

	return count
}

// TestPostVariations verifies that the list covers each thing the export's
// output contract is read for, and that every body in it is one the
// application would have accepted.
//
// The coverage is checked by property rather than by counting entries: what
// makes the list worth having is that a body of each kind is in it, and an
// entry deleted from it should fail here by name rather than by an off-by-one
// against a total.
//
// [Ja] TestPostVariations は、一覧がエクスポートの出力契約を読み合わせる観点を
// ひとつずつ覆っていること、そしてそこに書かれた本文がいずれもアプリケーションが
// 受け付けたはずのものであることを検証する。
//
// 覆っているかどうかは件数ではなく性質で確認する。この一覧に価値があるのは、
// それぞれの種類の本文が含まれていることによるものであり、1 件が削除されたときに
// 失敗すべきなのは、合計との 1 件のずれではなく、その観点の名前によってである。
func TestPostVariations(t *testing.T) {
	t.Parallel()

	for _, variation := range postVariations {
		// A body that is blank once trimmed is one the create validator
		// rejects, so the seed must not hold one: an archive is checked
		// against what an account can post.
		//
		// [Ja] トリムして空になる本文は作成時のバリデーションが拒否するもので
		// あるため、シードが持っていてはならない。アーカイブは、アカウントが
		// 投稿できるものに対して確認するものである。
		if strings.TrimSpace(variation.body) == "" {
			t.Errorf("本文 %q, want 空白のみでない本文", variation.body)
		}

		if length := utf8.RuneCountInString(variation.body); length > model.MaximumPostContentLength {
			t.Errorf("本文の文字数 = %d (%q), want %d 以下", length, variation.body, model.MaximumPostContentLength)
		}
	}

	tests := []struct {
		name  string
		holds func(postVariation) bool
	}{
		{
			name: "HTML の特殊文字を含む本文がある",
			holds: func(v postVariation) bool {
				return strings.Contains(v.body, "<") && strings.Contains(v.body, "&")
			},
		},
		{
			name:  "改行を含む本文がある",
			holds: func(v postVariation) bool { return strings.Contains(v.body, "\n") },
		},
		{
			name: "絵文字と CJK が混ざる本文がある",
			holds: func(v postVariation) bool {
				// Check the fixture's family emoji explicitly so that a
				// supplementary Han character cannot satisfy the emoji check.
				//
				// [Ja] 補助漢字が絵文字の検査を満たさないよう、フィクスチャの
				// 家族の絵文字を明示して確認する。
				return strings.Contains(v.body, "👨‍👩‍👧") &&
					strings.ContainsFunc(v.body, func(r rune) bool { return unicode.Is(unicode.Han, r) })
			},
		},
		{
			name:  "URL を含む本文がある",
			holds: func(v postVariation) bool { return strings.Contains(v.body, "https://") },
		},
		{
			name: "最大文字数ちょうどの本文がある",
			holds: func(v postVariation) bool {
				return utf8.RuneCountInString(v.body) == model.MaximumPostContentLength
			},
		},
		{
			name:  "1 文字だけの本文がある",
			holds: func(v postVariation) bool { return utf8.RuneCountInString(v.body) == 1 },
		},
		{
			name:  "前後に空白を持つ本文がある",
			holds: func(v postVariation) bool { return v.body != strings.TrimSpace(v.body) },
		},
		{
			name:  "削除されるポストがある",
			holds: func(v postVariation) bool { return v.discarded },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !slices.ContainsFunc(postVariations, tt.holds) {
				t.Errorf("postVariations に該当する本文が無い: %s", tt.name)
			}
		})
	}
}

// TestPostWriter_WritesVariationPosts verifies that every variation reaches
// the database as it was written, that the ones meant to be deleted are, that
// they are spread over the months before the run's own, and that the profile's
// newest post is the newest one a screen can still reach.
//
// [Ja] TestPostWriter_WritesVariationPosts は、それぞれの確認用ポストが書かれた
// とおりにデータベースへ届くこと、削除されるべきものが削除されること、実行自身の月
// より前の月々へ散らされること、そしてプロフィールの最も新しいポストが、画面から
// まだ辿り着ける中で最も新しいものであることを検証する。
func TestPostWriter_WritesVariationPosts(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	account := newVariationAccount(t, ctx, tx, now)
	writer := newVariationWriter(t, tx, now)

	if err := writer.writeVariationPosts(ctx, account); err != nil {
		t.Fatalf("エクスポート確認用ポストの作成に失敗: %v", err)
	}

	written := readVariationPosts(t, ctx, tx, account.profile.ID)
	if len(written) != len(postVariations) {
		t.Fatalf("ポスト数 = %d, want %d", len(written), len(postVariations))
	}

	// The bodies are compared as they came back rather than by a count, so
	// that a space or a newline the column or the driver dropped is reported
	// as the body it belongs to.
	//
	// [Ja] 本文は件数ではなく戻ってきたそのままの形で突き合わせる。カラムや
	// ドライバーが落とした空白や改行が、それが属する本文として報告されるように
	// するため。
	stored := make(map[string]storedPost, len(written))
	for _, post := range written {
		if _, found := stored[post.content]; found {
			t.Errorf("本文 %q が 2 件書き込まれている", post.content)
		}
		stored[post.content] = post
	}

	for _, variation := range postVariations {
		post, found := stored[variation.body]
		if !found {
			t.Errorf("本文 %q が書き込まれていない", variation.body)

			continue
		}

		if post.discarded != variation.discarded {
			t.Errorf("本文 %q の削除済み = %t, want %t", variation.body, post.discarded, variation.discarded)
		}
	}

	assertVariationMonths(t, account, written, now)
	assertLastPostAtIsNewestKept(t, ctx, tx, account, written)
}

// TestPostVariations_DiscardedAreLeftOutOfTheExport verifies that the deleted
// variations do not enter an export's snapshot, which is the whole reason for
// writing them.
//
// The check goes through ExportRepository.Create rather than through a
// predicate written here, because what has to hold is that the statement the
// application creates an export with leaves those posts out.
//
// [Ja] TestPostVariations_DiscardedAreLeftOutOfTheExport は、削除された確認用
// ポストがエクスポートの snapshot に入らないことを検証する。それらを書き込む理由の
// すべてがこれである。
//
// 検証をここに書いた条件式ではなく ExportRepository.Create を通して行うのは、
// 成り立つべきことが、アプリケーションがエクスポートを作成する文がそれらのポストを
// 除外することであるため。
func TestPostVariations_DiscardedAreLeftOutOfTheExport(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond)
	account := newVariationAccount(t, ctx, tx, now)
	writer := newVariationWriter(t, tx, now)

	if err := writer.writeVariationPosts(ctx, account); err != nil {
		t.Fatalf("エクスポート確認用ポストの作成に失敗: %v", err)
	}

	exports := repository.NewExportRepository(query.New(tx))
	export, err := exports.Create(ctx, repository.CreateExportInput{
		ProfileID: account.profile.ID,
		ActorID:   account.actor.ID,
	})
	if err != nil {
		t.Fatalf("エクスポートの作成に失敗: %v", err)
	}
	if export == nil {
		t.Fatal("エクスポートが作成されていない")
	}

	snapshot := readExportPostContents(t, ctx, tx, export.ID)

	var want []string
	for _, variation := range postVariations {
		if !variation.discarded {
			want = append(want, variation.body)
		}
	}

	slices.Sort(want)
	slices.Sort(snapshot)

	if !slices.Equal(snapshot, want) {
		t.Errorf("snapshot の本文 = %q, want %q", snapshot, want)
	}
}

// newVariationAccount creates the account the variations are written to. It
// goes through createAccounts rather than the builders so that the account
// holds the actor an export is requested by, and the time zone the months are
// counted in.
//
// [Ja] newVariationAccount は、確認用ポストの書き込み先となるアカウントを作成する。
// ビルダーではなく createAccounts を通すのは、エクスポートを要求する actor と、
// 月を数えるタイムゾーンを、そのアカウントが持つようにするため。
func newVariationAccount(t *testing.T, ctx context.Context, tx *sql.Tx, now time.Time) seedAccount {
	t.Helper()

	accounts, err := createAccounts(ctx, tx, newTestRoster(t), now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	account, err := accountForRole(accounts, roleMain)
	if err != nil {
		t.Fatalf("役割 %s のアカウントの取得に失敗: %v", roleMain, err)
	}

	return account
}

// newVariationWriter builds the writer the variations are written through.
//
// [Ja] newVariationWriter は、確認用ポストを書き込むための writer を組み立てる。
func newVariationWriter(t *testing.T, tx *sql.Tx, now time.Time) *postWriter {
	t.Helper()

	return newPostWriter(tx, testutil.NewOauthApplicationBuilder(t, tx).Build(), now)
}

// storedPost is one post read back out of the database, with whether it has
// been deleted.
//
// [Ja] storedPost は、データベースから読み戻したポスト 1 件と、それが削除済みで
// あるかどうか。
type storedPost struct {
	content     string
	publishedAt time.Time
	discarded   bool
}

// readVariationPosts returns the profile's posts, oldest first.
//
// [Ja] readVariationPosts は、プロフィールのポストを古いものから順に返す。
func readVariationPosts(t *testing.T, ctx context.Context, tx *sql.Tx, profileID model.ProfileID) []storedPost {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT content, published_at, discarded_at IS NOT NULL
		FROM posts
		WHERE profile_id = $1
		ORDER BY published_at
	`, uuid.UUID(profileID))
	if err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var posts []storedPost
	for rows.Next() {
		var post storedPost
		if err := rows.Scan(&post.content, &post.publishedAt, &post.discarded); err != nil {
			t.Fatalf("ポストの読み取りに失敗: %v", err)
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("ポストの取得に失敗: %v", err)
	}

	return posts
}

// readExportPostContents returns the bodies an export took a snapshot of.
//
// [Ja] readExportPostContents は、エクスポートが snapshot として取り込んだ本文を
// 返す。
func readExportPostContents(t *testing.T, ctx context.Context, tx *sql.Tx, exportID model.ExportID) []string {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT content FROM export_posts WHERE export_id = $1
	`, uuid.UUID(exportID))
	if err != nil {
		t.Fatalf("export_posts の取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var contents []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			t.Fatalf("export_posts の読み取りに失敗: %v", err)
		}
		contents = append(contents, content)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("export_posts の取得に失敗: %v", err)
	}

	return contents
}

// assertVariationMonths checks that the variations reach every month they are
// spread over and none of them lands in the month the run happens in.
//
// The run's own month is what a profile is opened at. A variation there would
// put a body written to be difficult at the top of that screen, and it would
// also make the account's newest post one of these rather than one of the
// everyday posts.
//
// [Ja] assertVariationMonths は、確認用ポストが散らされる月々のすべてに届くこと、
// そしていずれも実行が行われている月へ落ちないことを確認する。
//
// 実行自身の月は、プロフィールを開いたときに見える月である。そこに確認用ポストが
// あると、扱いにくくあるために書いた本文がその画面の先頭に来ることになり、また、
// アカウントの最も新しいポストが日常ポストではなくこれらのひとつになる。
func assertVariationMonths(t *testing.T, account seedAccount, posts []storedPost, now time.Time) {
	t.Helper()

	location, err := time.LoadLocation(account.user.TimeZone)
	if err != nil {
		t.Fatalf("タイムゾーン %q の読み込みに失敗: %v", account.user.TimeZone, err)
	}

	const monthKey = "2006-01"

	currentMonth := now.In(location).Format(monthKey)

	months := make(map[string]int)
	for _, post := range posts {
		month := post.publishedAt.In(location).Format(monthKey)
		if month == currentMonth {
			t.Errorf("本文 %q の月 = %s, want 実行が行われている月以外", post.content, month)
		}
		months[month]++
	}

	if len(months) != variationPostMonths {
		t.Errorf("ポストが分かれた月数 = %d (%v), want %d", len(months), months, variationPostMonths)
	}
}

// assertLastPostAtIsNewestKept checks that the profile records the newest post
// a screen can still reach, rather than a newer one that has been deleted.
//
// [Ja] assertLastPostAtIsNewestKept は、プロフィールが記録しているのが、画面から
// まだ辿り着ける中で最も新しいポストであり、それより新しい削除済みのポストでは
// ないことを確認する。
func assertLastPostAtIsNewestKept(
	t *testing.T,
	ctx context.Context,
	tx *sql.Tx,
	account seedAccount,
	posts []storedPost,
) {
	t.Helper()

	var newestKept time.Time
	for _, post := range posts {
		if !post.discarded && post.publishedAt.After(newestKept) {
			newestKept = post.publishedAt
		}
	}

	var lastPostAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `
		SELECT last_post_at FROM profiles WHERE id = $1
	`, uuid.UUID(account.profile.ID)).Scan(&lastPostAt); err != nil {
		t.Fatalf("プロフィールの last_post_at の取得に失敗: %v", err)
	}

	if !lastPostAt.Valid || !lastPostAt.Time.Equal(newestKept) {
		t.Errorf("last_post_at = %v, want %v", lastPostAt, newestKept)
	}

	if account.profile.LastPostAt == nil || !account.profile.LastPostAt.Equal(newestKept) {
		t.Errorf("モデルの LastPostAt = %v, want %v", account.profile.LastPostAt, newestKept)
	}
}
