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
}
