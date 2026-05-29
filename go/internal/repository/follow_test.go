package repository_test

import (
	"context"
	"testing"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
)

func TestFollowRepository_ListByTargetProfileID(t *testing.T) {
	t.Parallel()

	_, tx := testutil.SetupTx(t)
	ctx := context.Background()

	repo := repository.NewFollowRepository(testutil.QueriesWithTx(tx))

	t.Run("targetプロフィールをフォローしている関係を列挙できる", func(t *testing.T) {
		target := testutil.NewProfileBuilder(t, tx).Build()
		follower1 := testutil.NewProfileBuilder(t, tx).Build()
		follower2 := testutil.NewProfileBuilder(t, tx).Build()

		// target をフォローする 2 件
		testutil.NewFollowBuilder(t, tx).
			WithSourceProfileID(follower1).
			WithTargetProfileID(target).
			Build()
		testutil.NewFollowBuilder(t, tx).
			WithSourceProfileID(follower2).
			WithTargetProfileID(target).
			Build()

		// A follow where target is the source (target follows someone) must
		// not be returned.
		// [Ja] target が誰かをフォローしている関係 (source = target) は対象外。
		other := testutil.NewProfileBuilder(t, tx).Build()
		testutil.NewFollowBuilder(t, tx).
			WithSourceProfileID(target).
			WithTargetProfileID(other).
			Build()

		follows, err := repo.ListByTargetProfileID(ctx, target)
		if err != nil {
			t.Fatalf("ListByTargetProfileID() error = %v", err)
		}

		if len(follows) != 2 {
			t.Fatalf("len(follows) = %d, want 2", len(follows))
		}

		gotFollowers := map[model.ProfileID]bool{}
		for _, f := range follows {
			if f.TargetProfileID != target {
				t.Errorf("f.TargetProfileID = %v, want %v", f.TargetProfileID, target)
			}
			gotFollowers[f.SourceProfileID] = true
		}
		if !gotFollowers[follower1] || !gotFollowers[follower2] {
			t.Errorf("follower source profiles = %v, want both %v and %v", gotFollowers, follower1, follower2)
		}
	})

	t.Run("フォロワーがいない場合は空スライスを返す", func(t *testing.T) {
		target := testutil.NewProfileBuilder(t, tx).Build()

		follows, err := repo.ListByTargetProfileID(ctx, target)
		if err != nil {
			t.Fatalf("ListByTargetProfileID() error = %v", err)
		}
		if len(follows) != 0 {
			t.Errorf("len(follows) = %d, want 0", len(follows))
		}
	})
}
