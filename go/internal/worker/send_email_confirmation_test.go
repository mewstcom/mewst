package worker

import (
	"context"
	"testing"

	"github.com/riverqueue/river"

	"github.com/mewstcom/mewst/go/internal/email"
)

func TestSendEmailConfirmationWorker_Work(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    SendEmailConfirmationArgs
		wantErr bool
	}{
		{
			name: "正常系: 日本語ロケールでメール送信",
			args: SendEmailConfirmationArgs{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "ja",
			},
			wantErr: false,
		},
		{
			name: "正常系: 英語ロケールでメール送信",
			args: SendEmailConfirmationArgs{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "en",
			},
			wantErr: false,
		},
		{
			name: "正常系: 未知のロケールは日本語にフォールバック",
			args: SendEmailConfirmationArgs{
				Email:  "test@example.com",
				Code:   "123456",
				Locale: "unknown",
			},
			wantErr: false,
		},
		{
			name: "異常系: 空のメールアドレス",
			args: SendEmailConfirmationArgs{
				Email:  "",
				Code:   "123456",
				Locale: "ja",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sender := email.NewNoopSender()
			worker := NewSendEmailConfirmationWorker(sender)

			job := &river.Job[SendEmailConfirmationArgs]{
				Args: tt.args,
			}

			err := worker.Work(context.Background(), job)

			if (err != nil) != tt.wantErr {
				t.Errorf("Work() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// メールが送信されたことを確認（SendRawを使用しているためSentRawEmailsをチェック）
				if len(sender.SentRawEmails) != 1 {
					t.Errorf("SentRawEmails count = %v, want 1", len(sender.SentRawEmails))
					return
				}

				sentEmail := sender.SentRawEmails[0]
				if sentEmail.To != tt.args.Email {
					t.Errorf("SentEmail.To = %v, want %v", sentEmail.To, tt.args.Email)
				}

				// 件名を検証
				expectedSubject := "[Mewst] 確認用コード"
				if tt.args.Locale == "en" {
					expectedSubject = "[Mewst] Confirmation code"
				}
				if sentEmail.Subject != expectedSubject {
					t.Errorf("SentEmail.Subject = %v, want %v", sentEmail.Subject, expectedSubject)
				}
			}
		})
	}
}

func TestSendEmailConfirmationArgs_Kind(t *testing.T) {
	args := SendEmailConfirmationArgs{}
	if args.Kind() != "send_email_confirmation" {
		t.Errorf("Kind() = %v, want send_email_confirmation", args.Kind())
	}
}

func TestSendEmailConfirmationArgs_InsertOpts(t *testing.T) {
	args := SendEmailConfirmationArgs{}
	opts := args.InsertOpts()

	if opts.Queue != river.QueueDefault {
		t.Errorf("InsertOpts().Queue = %v, want %v", opts.Queue, river.QueueDefault)
	}

	if opts.MaxAttempts != 5 {
		t.Errorf("InsertOpts().MaxAttempts = %v, want 5", opts.MaxAttempts)
	}
}
