package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// mockJobInserter はテスト用のモック
type mockJobInserter struct {
	called bool
	args   river.JobArgs
	opts   *river.InsertOpts
	err    error
}

func (m *mockJobInserter) Insert(_ context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	m.opts = opts
	if m.err != nil {
		return nil, m.err
	}
	return &rivertype.JobInsertResult{}, nil
}

func TestEnqueueEmailConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		err := d.EnqueueEmailConfirmation(context.Background(), "test@example.com", "123456", "ja")
		if err != nil {
			t.Fatalf("EnqueueEmailConfirmation() error = %v", err)
		}

		if !mock.called {
			t.Fatal("Insert が呼ばれていません")
		}

		args, ok := mock.args.(SendEmailConfirmationArgs)
		if !ok {
			t.Fatalf("args の型が SendEmailConfirmationArgs ではありません: %T", mock.args)
		}
		if args.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", args.Email)
		}
		if args.Code != "123456" {
			t.Errorf("Code = %s, want 123456", args.Code)
		}
		if args.Locale != "ja" {
			t.Errorf("Locale = %s, want ja", args.Locale)
		}
		if mock.opts == nil {
			t.Fatal("InsertOpts が nil です")
		}
		if mock.opts.MaxAttempts != 5 {
			t.Errorf("MaxAttempts = %d, want 5", mock.opts.MaxAttempts)
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		err := d.EnqueueEmailConfirmation(context.Background(), "test@example.com", "123456", "ja")
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueEmailConfirmation() error = %v, want %v", err, insertErr)
		}
	})
}

func TestEnqueueFanoutPost(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		mock := &mockJobInserter{}
		d := NewDispatcher(mock)

		if err := d.EnqueueFanoutPost(context.Background(), "post-123"); err != nil {
			t.Fatalf("EnqueueFanoutPost() error = %v", err)
		}

		args, ok := mock.args.(FanoutPostArgs)
		if !ok {
			t.Fatalf("args の型が FanoutPostArgs ではありません: %T", mock.args)
		}
		if args.PostID != "post-123" {
			t.Errorf("PostID = %s, want post-123", args.PostID)
		}
		if mock.opts == nil || mock.opts.MaxAttempts != 5 {
			t.Errorf("InsertOpts が反映されていません: %+v", mock.opts)
		}
	})

	t.Run("異常系: inserter がエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		mock := &mockJobInserter{err: insertErr}
		d := NewDispatcher(mock)

		if err := d.EnqueueFanoutPost(context.Background(), "post-123"); !errors.Is(err, insertErr) {
			t.Errorf("EnqueueFanoutPost() error = %v, want %v", err, insertErr)
		}
	})
}

func TestEnqueueAddPostToTimeline(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	d := NewDispatcher(mock)

	if err := d.EnqueueAddPostToTimeline(context.Background(), "profile-123", "post-456"); err != nil {
		t.Fatalf("EnqueueAddPostToTimeline() error = %v", err)
	}

	args, ok := mock.args.(AddPostToTimelineArgs)
	if !ok {
		t.Fatalf("args の型が AddPostToTimelineArgs ではありません: %T", mock.args)
	}
	if args.ProfileID != "profile-123" {
		t.Errorf("ProfileID = %s, want profile-123", args.ProfileID)
	}
	if args.PostID != "post-456" {
		t.Errorf("PostID = %s, want post-456", args.PostID)
	}
}

func TestFanoutPostArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (FanoutPostArgs{}).Kind(); got != "fanout_post" {
		t.Errorf("Kind() = %s, want fanout_post", got)
	}
}

func TestAddPostToTimelineArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (AddPostToTimelineArgs{}).Kind(); got != "add_post_to_timeline" {
		t.Errorf("Kind() = %s, want add_post_to_timeline", got)
	}
}

func TestDeferredInserter_DelegatesAfterSetInserter(t *testing.T) {
	t.Parallel()

	mock := &mockJobInserter{}
	deferred := &DeferredInserter{}
	deferred.SetInserter(mock)

	// The Dispatcher built around the DeferredInserter must reach the wired
	// inserter once SetInserter has been called.
	// [Ja] DeferredInserter を包んだ Dispatcher は、SetInserter 後に注入済みの
	// inserter へ到達できなければならない。
	d := NewDispatcher(deferred)
	if err := d.EnqueueFanoutPost(context.Background(), "post-123"); err != nil {
		t.Fatalf("EnqueueFanoutPost() error = %v", err)
	}

	if !mock.called {
		t.Fatal("注入した inserter の Insert が呼ばれていません")
	}
}

func TestSendEmailConfirmationArgs_Kind(t *testing.T) {
	t.Parallel()
	if got := (SendEmailConfirmationArgs{}).Kind(); got != "send_email_confirmation" {
		t.Errorf("Kind() = %s, want send_email_confirmation", got)
	}
}

func TestSendEmailConfirmationArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	opts := (SendEmailConfirmationArgs{}).InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
	// Email is demoted to Priority 2 so the timeline delivery jobs (kept at the
	// default top priority) are worked first when the pool is saturated.
	// [Ja] メールは Priority 2 に降格し、プール飽和時にタイムライン配信ジョブ (既定の
	// 最高優先度のまま) を先に処理させる。
	if opts.Priority != 2 {
		t.Errorf("InsertOpts().Priority = %v, want 2 (demoted below timeline delivery jobs)", opts.Priority)
	}
}

func TestFanoutPostArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	opts := (FanoutPostArgs{}).InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
	// Priority is left unset (0), which River treats as the default top priority
	// (1). It must outrank the demoted email job so fanout is fetched first.
	// [Ja] Priority は未設定 (0) で、River はこれを既定の最高優先度 (1) として扱う。
	// 降格したメールジョブより上位であり、fanout が先に fetch される必要がある。
	if opts.Priority != 0 {
		t.Errorf("InsertOpts().Priority = %v, want 0 (default top priority)", opts.Priority)
	}
	if emailOpts := (SendEmailConfirmationArgs{}).InsertOpts(); opts.Priority >= emailOpts.Priority {
		t.Errorf("fanout priority (%d) must outrank email priority (%d)", opts.Priority, emailOpts.Priority)
	}
}

func TestAddPostToTimelineArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	opts := (AddPostToTimelineArgs{}).InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
	// Priority is left unset (0 = default top priority), keeping per-follower
	// delivery ahead of the demoted email job (Priority 2).
	// [Ja] Priority は未設定 (0 = 既定の最高優先度) で、フォロワー単位の配信を降格した
	// メールジョブ (Priority 2) より上位に保つ。
	if opts.Priority != 0 {
		t.Errorf("InsertOpts().Priority = %v, want 0 (default top priority)", opts.Priority)
	}
	if emailOpts := (SendEmailConfirmationArgs{}).InsertOpts(); opts.Priority >= emailOpts.Priority {
		t.Errorf("add_post priority (%d) must outrank email priority (%d)", opts.Priority, emailOpts.Priority)
	}
}
