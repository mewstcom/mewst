package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

const (
	// benchPostCountとbenchPostContentRunesはエクスポートの要件が想定する
	// プロフィール (最大長の投稿を長期間持つプロフィール) を表す。
	// benchPostIntervalはそれを約3年へ分散させ、実際のアーカイブが分割される月と
	// 同じ形でsnapshotを走査できるようにする。
	benchPostCount        = 100_000
	benchPostContentRunes = 160
	benchPostInterval     = 15 * time.Minute
)

// benchPostsPublishedFromはベンチマークのフィクスチャの投稿が始まる時刻。
var benchPostsPublishedFrom = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

// BenchmarkExportRepository_Createはエクスポート申請リクエストの中で起きる
// snapshotのmaterializeを計測する。Createはexportを挿入するのと同じ文で
// プロフィールのkeptな投稿を複製するため。snapshot 1件が増やすストレージも
// 時間と併せて報告し、ユーザーが待つコストとDBが負うコストを同時に判断できる
// ようにする。
func BenchmarkExportRepository_Create(b *testing.B) {
	_, tx := testutil.SetupTx(b)
	ctx := context.Background()

	profileID, actorID := newBenchExportFixture(b, tx)
	repo := repository.NewExportRepository(testutil.QueriesWithTx(tx))
	sizeBefore := exportPostsBytes(b, tx)

	var exports int64
	for b.Loop() {
		export, err := repo.Create(ctx, repository.CreateExportInput{ProfileID: profileID, ActorID: actorID})
		if err != nil {
			b.Fatalf("エクスポートの作成に失敗: %v", err)
		}
		exports++

		// 部分ユニークインデックスはプロフィールごとに進行中のエクスポートを
		// 1件しか許さないため、次の反復の前に作成したものを終端状態まで進める。
		b.StopTimer()
		failExportSnapshot(b, tx, export)
		b.StartTimer()
	}

	b.ReportMetric(float64(exportPostsBytes(b, tx)-sizeBefore)/float64(exports)/(1<<20), "snapshot_MiB")
}

// BenchmarkExportPostRepository_ListByExportIDInRangeはsnapshot全体を
// 1ページずつ走査する処理を計測する。生成処理が投稿を読む形と同じため。各ケースは
// 同じsnapshotをページサイズだけ変えて走査するので、ページサイズが要する
// ラウンドトリップ数と、1ページが保持するメモリを比較できる。
func BenchmarkExportPostRepository_ListByExportIDInRange(b *testing.B) {
	_, tx := testutil.SetupTx(b)
	ctx := context.Background()

	profileID, actorID := newBenchExportFixture(b, tx)
	export := createExportSnapshot(b, tx, profileID, actorID)
	repo := repository.NewExportPostRepository(testutil.QueriesWithTx(tx))

	location := mustLoadLocation(b, "Asia/Tokyo")
	months, err := repo.ListMonthsByExportID(ctx, repository.ListExportPostMonthsByExportIDInput{
		ExportID: export.ID,
		Location: location,
	})
	if err != nil {
		b.Fatalf("月一覧の取得に失敗: %v", err)
	}

	for _, pageSize := range []int32{100, 500, 1_000, 5_000} {
		b.Run(fmt.Sprintf("page=%d", pageSize), func(b *testing.B) {
			var pages, posts int64
			for b.Loop() {
				pages, posts = walkBenchSnapshot(b, ctx, repo, export.ID, months, pageSize)
				if posts != benchPostCount {
					b.Errorf("走査した投稿数 = %d、期待値 = %d", posts, benchPostCount)
				}
			}

			b.ReportMetric(float64(pages), "pages")
		})
	}
}

// walkBenchSnapshotはsnapshotの全ての月をページ送りで走査し、読み取った
// ページ数と投稿件数を返す。
func walkBenchSnapshot(
	b *testing.B,
	ctx context.Context,
	repo *repository.ExportPostRepository,
	exportID model.ExportID,
	months []repository.PostMonth,
	pageSize int32,
) (pages, posts int64) {
	b.Helper()

	for _, month := range months {
		var cursor *repository.PostCursor
		for {
			page, next, err := repo.ListByExportIDInRange(ctx, repository.ListExportPostsByExportIDInRangeInput{
				ExportID: exportID,
				Month:    month,
				Cursor:   cursor,
				PageSize: pageSize,
			})
			if err != nil {
				b.Fatalf("投稿ページの取得に失敗: %v", err)
			}
			pages++
			posts += int64(len(page))
			cursor = next
			if cursor == nil {
				break
			}
		}
	}
	return pages, posts
}

// newBenchExportFixtureは最大長の投稿をbenchPostCount件持つエクスポート
// 対象を作成する。
func newBenchExportFixture(b *testing.B, tx *sql.Tx) (model.ProfileID, model.ActorID) {
	b.Helper()

	owner := testutil.NewProfileOwner(b, tx)
	profileID, actorID := owner.ProfileID, owner.ActorID
	testutil.NewPostBuilder(b, tx).
		WithProfileID(profileID).
		WithOauthApplicationID(testutil.NewOauthApplicationBuilder(b, tx).Build()).
		WithContent(strings.Repeat("あ", benchPostContentRunes)).
		WithPublishedAt(benchPostsPublishedFrom).
		BuildMany(benchPostCount, benchPostInterval)

	return profileID, actorID
}

// exportPostsBytesはexport_postsが占めるストレージを、インデックスと
// TOAST領域を含めて返す。ベンチマークはテストDBに対して単独で実行するため、
// 差分はすべて自身が作成したsnapshotのもの。
func exportPostsBytes(b *testing.B, tx *sql.Tx) int64 {
	b.Helper()

	var size int64
	if err := tx.QueryRow(`SELECT pg_total_relation_size('export_posts')`).Scan(&size); err != nil {
		b.Fatalf("export_postsのサイズ取得に失敗: %v", err)
	}
	return size
}
