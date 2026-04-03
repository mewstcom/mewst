package dispatcher_test

import (
	"context"
	"errors"
	"testing"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mewstcom/mewst/go/internal/dispatcher"
)

// mockInserter はテスト用のモック inserter
type mockInserter struct {
	called bool
	args   river.JobArgs
	err    error
}

func (m *mockInserter) Insert(_ context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	m.called = true
	m.args = args
	if m.err != nil {
		return nil, m.err
	}
	return &rivertype.JobInsertResult{}, nil
}

func TestDispatcher_EnqueueEmailConfirmation(t *testing.T) {
	t.Parallel()

	t.Run("正常系: ジョブをエンキューできる", func(t *testing.T) {
		t.Parallel()

		inserter := &mockInserter{}
		d := dispatcher.NewDispatcher(inserter)

		args := dispatcher.SendEmailConfirmationArgs{
			Email:  "test@example.com",
			Code:   "123456",
			Locale: "ja",
		}

		err := d.EnqueueEmailConfirmation(context.Background(), args)
		if err != nil {
			t.Fatalf("EnqueueEmailConfirmation() error = %v", err)
		}

		if !inserter.called {
			t.Fatal("Insert() が呼ばれていません")
		}

		emailArgs, ok := inserter.args.(dispatcher.SendEmailConfirmationArgs)
		if !ok {
			t.Fatalf("args の型が SendEmailConfirmationArgs ではありません: %T", inserter.args)
		}
		if emailArgs.Email != "test@example.com" {
			t.Errorf("Email = %s, want test@example.com", emailArgs.Email)
		}
		if emailArgs.Code != "123456" {
			t.Errorf("Code = %s, want 123456", emailArgs.Code)
		}
		if emailArgs.Locale != "ja" {
			t.Errorf("Locale = %s, want ja", emailArgs.Locale)
		}
	})

	t.Run("異常系: inserterがエラーを返す", func(t *testing.T) {
		t.Parallel()

		insertErr := errors.New("エンキューエラー")
		inserter := &mockInserter{err: insertErr}
		d := dispatcher.NewDispatcher(inserter)

		err := d.EnqueueEmailConfirmation(context.Background(), dispatcher.SendEmailConfirmationArgs{
			Email:  "test@example.com",
			Code:   "123456",
			Locale: "ja",
		})
		if !errors.Is(err, insertErr) {
			t.Errorf("EnqueueEmailConfirmation() error = %v, want %v", err, insertErr)
		}
	})
}

func TestSendEmailConfirmationArgs_Kind(t *testing.T) {
	t.Parallel()

	args := dispatcher.SendEmailConfirmationArgs{}
	if got := args.Kind(); got != "send_email_confirmation" {
		t.Errorf("Kind() = %v, want send_email_confirmation", got)
	}
}

func TestSendEmailConfirmationArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := dispatcher.SendEmailConfirmationArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}

	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
}
