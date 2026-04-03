// Package dispatcher はジョブキューへの投入を抽象化し、UseCase ↔ Worker 間の循環依存を解消します
package dispatcher

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// JobInserter はジョブをキューに追加するインターフェース
type JobInserter interface {
	Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error)
}

// Dispatcher はジョブのエンキューを管理する
type Dispatcher struct {
	inserter JobInserter
}

// NewDispatcher は新しいDispatcherを作成する
func NewDispatcher(inserter JobInserter) *Dispatcher {
	return &Dispatcher{inserter: inserter}
}

// EnqueueEmailConfirmation はメール確認コード送信ジョブをエンキューする
func (d *Dispatcher) EnqueueEmailConfirmation(ctx context.Context, args SendEmailConfirmationArgs) error {
	_, err := d.inserter.Insert(ctx, args)
	return err
}
