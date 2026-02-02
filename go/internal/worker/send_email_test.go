package worker

import (
	"context"
	"testing"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/internal/email"
)

func TestSendEmailArgs_Kind(t *testing.T) {
	t.Parallel()

	args := SendEmailArgs{}
	if got := args.Kind(); got != "send_email" {
		t.Errorf("Kind() = %v, want send_email", got)
	}
}

func TestSendEmailArgs_InsertOpts(t *testing.T) {
	t.Parallel()

	args := SendEmailArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}

	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
}

func TestSendEmailWorker_Work(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	noopSender := email.NewNoopSender()
	w := NewSendEmailWorker(noopSender)

	job := &river.Job[SendEmailArgs]{
		Args: SendEmailArgs{
			To:       "test@example.com",
			Subject:  "Test Subject",
			HTMLBody: "<p>HTML Body</p>",
			TextBody: "Text Body",
		},
	}

	err := w.Work(ctx, job)
	if err != nil {
		t.Fatalf("Work() error = %v", err)
	}

	if len(noopSender.SentRawEmails) != 1 {
		t.Fatalf("SentRawEmails count = %v, want 1", len(noopSender.SentRawEmails))
	}

	sentEmail := noopSender.SentRawEmails[0]
	if sentEmail.To != "test@example.com" {
		t.Errorf("To = %v, want test@example.com", sentEmail.To)
	}
	if sentEmail.Subject != "Test Subject" {
		t.Errorf("Subject = %v, want Test Subject", sentEmail.Subject)
	}
	if sentEmail.HTMLBody != "<p>HTML Body</p>" {
		t.Errorf("HTMLBody = %v, want <p>HTML Body</p>", sentEmail.HTMLBody)
	}
	if sentEmail.TextBody != "Text Body" {
		t.Errorf("TextBody = %v, want Text Body", sentEmail.TextBody)
	}
}
