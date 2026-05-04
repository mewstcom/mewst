package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestRateLimitRepository_Increment(t *testing.T) {
	t.Parallel()

	t.Run("新規レコードを作成できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := repository.NewRateLimitRepository(testutil.QueriesWithTx(tx))
		ctx := context.Background()

		now := time.Now().UTC()
		windowStart := now.Truncate(time.Hour)

		result, err := repo.Increment(ctx, repository.IncrementInput{
			Key:         "test:increment_new",
			WindowStart: windowStart,
		})
		if err != nil {
			t.Fatalf("Increment() error = %v", err)
		}

		if result.Count != 1 {
			t.Errorf("Count = %d, want 1", result.Count)
		}
	})

	t.Run("既存レコードのカウントをインクリメントできる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := repository.NewRateLimitRepository(testutil.QueriesWithTx(tx))
		ctx := context.Background()

		now := time.Now().UTC()
		windowStart := now.Truncate(time.Hour)

		// 1回目のインクリメント
		_, err := repo.Increment(ctx, repository.IncrementInput{
			Key:         "test:increment_existing",
			WindowStart: windowStart,
		})
		if err != nil {
			t.Fatalf("1回目のIncrement() error = %v", err)
		}

		// 2回目のインクリメント
		result, err := repo.Increment(ctx, repository.IncrementInput{
			Key:         "test:increment_existing",
			WindowStart: windowStart,
		})
		if err != nil {
			t.Fatalf("2回目のIncrement() error = %v", err)
		}

		if result.Count != 2 {
			t.Errorf("Count = %d, want 2", result.Count)
		}
	})

	t.Run("異なるキーは別々にカウントされる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := repository.NewRateLimitRepository(testutil.QueriesWithTx(tx))
		ctx := context.Background()

		now := time.Now().UTC()
		windowStart := now.Truncate(time.Hour)

		// key1を2回インクリメント
		for i := 0; i < 2; i++ {
			_, err := repo.Increment(ctx, repository.IncrementInput{
				Key:         "test:diff_key1",
				WindowStart: windowStart,
			})
			if err != nil {
				t.Fatalf("key1のIncrement() error = %v", err)
			}
		}

		// key2を1回インクリメント
		result, err := repo.Increment(ctx, repository.IncrementInput{
			Key:         "test:diff_key2",
			WindowStart: windowStart,
		})
		if err != nil {
			t.Fatalf("key2のIncrement() error = %v", err)
		}

		if result.Count != 1 {
			t.Errorf("key2のCount = %d, want 1", result.Count)
		}
	})
}

func TestRateLimitRepository_DeleteOldRecords(t *testing.T) {
	t.Parallel()

	t.Run("古いレコードを削除できる", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		repo := repository.NewRateLimitRepository(testutil.QueriesWithTx(tx))
		ctx := context.Background()

		now := time.Now().UTC()
		windowStart := now.Truncate(time.Hour)

		// レコードを作成
		_, err := repo.Increment(ctx, repository.IncrementInput{
			Key:         "test:delete_old",
			WindowStart: windowStart,
		})
		if err != nil {
			t.Fatalf("Increment() error = %v", err)
		}

		// 現在より未来のcutoffで削除（削除されない）
		err = repo.DeleteOldRecords(ctx, now.Add(2*time.Hour))
		if err != nil {
			t.Errorf("DeleteOldRecords() error = %v", err)
		}
	})
}
