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

// discardedVariationCountは、エクスポート確認用ポストのうち、書き込んだ
// あとに削除するものの件数。書き下さず一覧から数えるのは、一覧へ1件足したときに
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

// TestPostVariationsは、一覧がエクスポートの出力契約を読み合わせる観点を
// ひとつずつ覆っていること、そしてそこに書かれた本文がいずれもアプリケーションが
// 受け付けたはずのものであることを検証する。
//
// 覆っているかどうかは件数ではなく性質で確認する。この一覧に価値があるのは、
// それぞれの種類の本文が含まれていることによるものであり、1件が削除されたときに
// 失敗すべきなのは、合計との1件のずれではなく、その観点の名前によってである。
func TestPostVariations(t *testing.T) {
	t.Parallel()

	for _, variation := range postVariations {
		// トリムして空になる本文は作成時のバリデーションが拒否するもので
		// あるため、シードが持っていてはならない。アーカイブは、アカウントが
		// 投稿できるものに対して確認するものである。
		if strings.TrimSpace(variation.body) == "" {
			t.Errorf("本文%q、期待値 = 空白のみでない本文", variation.body)
		}

		if length := utf8.RuneCountInString(variation.body); length > model.MaximumPostContentLength {
			t.Errorf("本文の文字数 = %d (%q)、期待値 = %d以下", length, variation.body, model.MaximumPostContentLength)
		}
	}

	tests := []struct {
		name  string
		holds func(postVariation) bool
	}{
		{
			name: "HTMLの特殊文字を含む本文がある",
			holds: func(v postVariation) bool {
				return strings.Contains(v.body, "<") && strings.Contains(v.body, "&")
			},
		},
		{
			name:  "改行を含む本文がある",
			holds: func(v postVariation) bool { return strings.Contains(v.body, "\n") },
		},
		{
			name: "絵文字とCJKが混ざる本文がある",
			holds: func(v postVariation) bool {
				// 補助漢字が絵文字の検査を満たさないよう、フィクスチャの
				// 家族の絵文字を明示して確認する。
				return strings.Contains(v.body, "👨‍👩‍👧") &&
					strings.ContainsFunc(v.body, func(r rune) bool { return unicode.Is(unicode.Han, r) })
			},
		},
		{
			name:  "URLを含む本文がある",
			holds: func(v postVariation) bool { return strings.Contains(v.body, "https://") },
		},
		{
			name: "最大文字数ちょうどの本文がある",
			holds: func(v postVariation) bool {
				return utf8.RuneCountInString(v.body) == model.MaximumPostContentLength
			},
		},
		{
			name:  "1文字だけの本文がある",
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
				t.Errorf("postVariationsに該当する本文が無い: %s", tt.name)
			}
		})
	}
}

// TestPostWriter_WritesVariationPostsは、それぞれの確認用ポストが書かれた
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
		t.Fatalf("ポスト数 = %d、期待値 = %d", len(written), len(postVariations))
	}

	// 本文は件数ではなく戻ってきたそのままの形で突き合わせる。カラムや
	// ドライバーが落とした空白や改行が、それが属する本文として報告されるように
	// するため。
	stored := make(map[string]storedPost, len(written))
	for _, post := range written {
		if _, found := stored[post.content]; found {
			t.Errorf("本文%qが2件書き込まれている", post.content)
		}
		stored[post.content] = post
	}

	for _, variation := range postVariations {
		post, found := stored[variation.body]
		if !found {
			t.Errorf("本文%qが書き込まれていない", variation.body)

			continue
		}

		if post.discarded != variation.discarded {
			t.Errorf("本文%qの削除済み = %t、期待値 = %t", variation.body, post.discarded, variation.discarded)
		}
	}

	assertVariationMonths(t, account, written, now)
	assertLastPostAtIsNewestKept(t, ctx, tx, account, written)
}

// TestPostVariations_DiscardedAreLeftOutOfTheExportは、削除された確認用
// ポストがエクスポートのsnapshotに入らないことを検証する。それらを書き込む理由の
// すべてがこれである。
//
// 検証をここに書いた条件式ではなくExportRepository.Createを通して行うのは、
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
		t.Errorf("snapshotの本文 = %q、期待値 = %q", snapshot, want)
	}
}

// newVariationAccountは、確認用ポストの書き込み先となるアカウントを作成する。
// ビルダーではなくcreateAccountsを通すのは、エクスポートを要求するactorと、
// 月を数えるタイムゾーンを、そのアカウントが持つようにするため。
func newVariationAccount(t *testing.T, ctx context.Context, tx *sql.Tx, now time.Time) seedAccount {
	t.Helper()

	accounts, err := createAccounts(ctx, tx, newTestRoster(t), now)
	if err != nil {
		t.Fatalf("アカウントの作成に失敗: %v", err)
	}

	account, err := accountForRole(accounts, roleMain)
	if err != nil {
		t.Fatalf("役割%sのアカウントの取得に失敗: %v", roleMain, err)
	}

	return account
}

// newVariationWriterは、確認用ポストを書き込むためのwriterを組み立てる。
func newVariationWriter(t *testing.T, tx *sql.Tx, now time.Time) *postWriter {
	t.Helper()

	return newPostWriter(tx, testutil.NewOauthApplicationBuilder(t, tx).Build(), now)
}

// storedPostは、データベースから読み戻したポスト1件と、それが削除済みで
// あるかどうか。
type storedPost struct {
	content     string
	publishedAt time.Time
	discarded   bool
}

// readVariationPostsは、プロフィールのポストを古いものから順に返す。
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

// readExportPostContentsは、エクスポートがsnapshotとして取り込んだ本文を
// 返す。
func readExportPostContents(t *testing.T, ctx context.Context, tx *sql.Tx, exportID model.ExportID) []string {
	t.Helper()

	rows, err := tx.QueryContext(ctx, `
		SELECT content FROM export_posts WHERE export_id = $1
	`, uuid.UUID(exportID))
	if err != nil {
		t.Fatalf("export_postsの取得に失敗: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var contents []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			t.Fatalf("export_postsの読み取りに失敗: %v", err)
		}
		contents = append(contents, content)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("export_postsの取得に失敗: %v", err)
	}

	return contents
}

// assertVariationMonthsは、確認用ポストが散らされる月々のすべてに届くこと、
// そしていずれも実行が行われている月へ落ちないことを確認する。
//
// 実行自身の月は、プロフィールを開いたときに見える月である。そこに確認用ポストが
// あると、扱いにくくあるために書いた本文がその画面の先頭に来ることになり、また、
// アカウントの最も新しいポストが日常ポストではなくこれらのひとつになる。
func assertVariationMonths(t *testing.T, account seedAccount, posts []storedPost, now time.Time) {
	t.Helper()

	location, err := time.LoadLocation(account.user.TimeZone)
	if err != nil {
		t.Fatalf("タイムゾーン%qの読み込みに失敗: %v", account.user.TimeZone, err)
	}

	const monthKey = "2006-01"

	currentMonth := now.In(location).Format(monthKey)

	months := make(map[string]int)
	for _, post := range posts {
		month := post.publishedAt.In(location).Format(monthKey)
		if month == currentMonth {
			t.Errorf("本文%qの月 = %s、期待値 = 実行が行われている月以外", post.content, month)
		}
		months[month]++
	}

	if len(months) != variationPostMonths {
		t.Errorf("ポストが分かれた月数 = %d (%v)、期待値 = %d", len(months), months, variationPostMonths)
	}
}

// assertLastPostAtIsNewestKeptは、プロフィールが記録しているのが、画面から
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
		t.Fatalf("プロフィールのlast_post_atの取得に失敗: %v", err)
	}

	if !lastPostAt.Valid || !lastPostAt.Time.Equal(newestKept) {
		t.Errorf("last_post_at = %v、期待値 = %v", lastPostAt, newestKept)
	}

	if account.profile.LastPostAt == nil || !account.profile.LastPostAt.Equal(newestKept) {
		t.Errorf("モデルのLastPostAt = %v、期待値 = %v", account.profile.LastPostAt, newestKept)
	}
}
