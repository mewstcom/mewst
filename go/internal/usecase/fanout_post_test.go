package usecase_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/repository"
	"github.com/mewstcom/mewst/go/internal/testutil"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// recordingJobInserterはenqueueされた全ジョブを記録し、fanoutがAddPostToTimeline
// ジョブを何件・どの引数で出したかをテストで検証できるようにする。
type recordingJobInserter struct {
	inserts []river.JobArgs
	err     error
}

func (m *recordingJobInserter) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.inserts = append(m.inserts, args)
	return &rivertype.JobInsertResult{}, nil
}

func TestFanoutPostUsecase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("各フォロワーにAddPostToTimelineジョブをenqueueする", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		authorID := testutil.NewProfileBuilder(t, tx).Build()
		oauthAppID := testutil.NewOauthApplicationBuilder(t, tx).Build()
		postID := testutil.NewPostBuilder(t, tx).
			WithProfileID(authorID).
			WithOauthApplicationID(oauthAppID).
			Build()

		follower1 := testutil.NewProfileBuilder(t, tx).Build()
		follower2 := testutil.NewProfileBuilder(t, tx).Build()
		testutil.NewFollowBuilder(t, tx).WithSourceProfileID(follower1).WithTargetProfileID(authorID).Build()
		testutil.NewFollowBuilder(t, tx).WithSourceProfileID(follower2).WithTargetProfileID(authorID).Build()

		postRepo := repository.NewPostRepository(testutil.QueriesWithTx(tx))
		followRepo := repository.NewFollowRepository(testutil.QueriesWithTx(tx))
		mock := &recordingJobInserter{}
		uc := usecase.NewFanoutPostUsecase(postRepo, followRepo, dispatcher.NewDispatcher(mock))

		if err := uc.Execute(ctx, usecase.FanoutPostInput{PostID: postID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}

		if len(mock.inserts) != 2 {
			t.Fatalf("enqueue件数 = %d、期待値 = 2", len(mock.inserts))
		}

		enqueuedProfiles := map[string]bool{}
		for _, a := range mock.inserts {
			args, ok := a.(dispatcher.AddPostToTimelineArgs)
			if !ok {
				t.Fatalf("argsの型がAddPostToTimelineArgsではありません: %T", a)
			}
			if args.PostID != postID.String() {
				t.Errorf("PostID = %s、期待値 = %s", args.PostID, postID.String())
			}
			enqueuedProfiles[args.ProfileID] = true
		}
		if !enqueuedProfiles[follower1.String()] || !enqueuedProfiles[follower2.String()] {
			t.Errorf("フォロワー宛のジョブが揃っていません: %v", enqueuedProfiles)
		}
	})

	t.Run("フォロワーがいなければ何もenqueueしない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		authorID := testutil.NewProfileBuilder(t, tx).Build()
		oauthAppID := testutil.NewOauthApplicationBuilder(t, tx).Build()
		postID := testutil.NewPostBuilder(t, tx).
			WithProfileID(authorID).
			WithOauthApplicationID(oauthAppID).
			Build()

		postRepo := repository.NewPostRepository(testutil.QueriesWithTx(tx))
		followRepo := repository.NewFollowRepository(testutil.QueriesWithTx(tx))
		mock := &recordingJobInserter{}
		uc := usecase.NewFanoutPostUsecase(postRepo, followRepo, dispatcher.NewDispatcher(mock))

		if err := uc.Execute(ctx, usecase.FanoutPostInput{PostID: postID}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if len(mock.inserts) != 0 {
			t.Errorf("enqueue件数 = %d、期待値 = 0", len(mock.inserts))
		}
	})

	t.Run("投稿が存在しなければエラーなく何もenqueueしない", func(t *testing.T) {
		t.Parallel()

		_, tx := testutil.SetupTx(t)
		ctx := context.Background()

		postRepo := repository.NewPostRepository(testutil.QueriesWithTx(tx))
		followRepo := repository.NewFollowRepository(testutil.QueriesWithTx(tx))
		mock := &recordingJobInserter{}
		uc := usecase.NewFanoutPostUsecase(postRepo, followRepo, dispatcher.NewDispatcher(mock))

		// 存在しないランダムな投稿ID。
		if err := uc.Execute(ctx, usecase.FanoutPostInput{PostID: model.PostID(uuid.New())}); err != nil {
			t.Fatalf("Execute()のエラー = %v", err)
		}
		if len(mock.inserts) != 0 {
			t.Errorf("enqueue件数 = %d、期待値 = 0", len(mock.inserts))
		}
	})
}
