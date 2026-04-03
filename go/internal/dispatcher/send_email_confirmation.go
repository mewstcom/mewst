package dispatcher

import "github.com/riverqueue/river"

// SendEmailConfirmationArgs はメール確認コード送信ジョブの引数
type SendEmailConfirmationArgs struct {
	Email  string `json:"email"`
	Code   string `json:"code"`
	Locale string `json:"locale"`
}

// Kind はジョブの種類を返す（riverのインターフェース実装）
func (SendEmailConfirmationArgs) Kind() string {
	return "send_email_confirmation"
}

// InsertOpts はジョブのオプションを返す（riverのインターフェース実装）
func (SendEmailConfirmationArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       river.QueueDefault,
		MaxAttempts: 5, // 最大5回リトライ
	}
}
