package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/query"
)

// exportProfileLockReleaseTimeoutは、操作のcontextから切り離したlock解放の
// 上限時間。キャンセルされたuploadでも、session単位のlockをコネクションプールへ
// 返す前に解放する必要がある。
const exportProfileLockReleaseTimeout = 5 * time.Second

// ExportProfileDeletionGuardRepositoryはexport操作とプロフィール削除を
// 調整する。永続マーカーで新しい処理を閉じ、PostgreSQL advisory lockにより、外部
// I/Oの間に通常transactionを保持せず、削除が境界を越えた処理を待てるようにする。
type ExportProfileDeletionGuardRepository struct {
	db *sql.DB
	q  *query.Queries
}

// NewExportProfileDeletionGuardRepositoryは
// ExportProfileDeletionGuardRepositoryを生成する。
func NewExportProfileDeletionGuardRepository(db *sql.DB) *ExportProfileDeletionGuardRepository {
	return &ExportProfileDeletionGuardRepository{
		db: db,
		q:  query.New(db),
	}
}

// BeginOperationは永続的な削除マーカーを確認してからプロフィールの操作用
// 共有lockを待ち、取得後にもう一度確認する。最初の確認により、削除開始後に投入
// された処理は、削除用の排他lockの後ろで待たず即座に停止する。2回目の確認は、
// 最初の確認とlock取得の間に削除が始まる競合を閉じる。allowed=trueの場合、
// 返されたrelease関数を必ず呼ぶ。存在しない、または削除中のプロフィールでは
// allowed=falseを返す。
func (r *ExportProfileDeletionGuardRepository) BeginOperation(
	ctx context.Context,
	profileID model.ProfileID,
) (release func() error, allowed bool, err error) {
	allowed, err = exportProfileOperationAllowed(ctx, r.q, profileID)
	if err != nil {
		return nil, false, fmt.Errorf("プロフィールのexport操作可否の事前確認に失敗: %w", err)
	}
	if !allowed {
		return nil, false, nil
	}

	lock, err := r.acquire(ctx, profileID, true)
	if err != nil {
		return nil, false, err
	}

	allowed, checkErr := exportProfileOperationAllowed(ctx, lock.q, profileID)
	if checkErr != nil {
		releaseErr := lock.release()
		return nil, false, errors.Join(
			fmt.Errorf("プロフィールのexport操作可否の確認に失敗: %w", checkErr),
			releaseErr,
		)
	}
	if !allowed {
		if err := lock.release(); err != nil {
			return nil, false, fmt.Errorf("export操作用プロフィールlockの解放に失敗: %w", err)
		}
		return nil, false, nil
	}
	return lock.release, true, nil
}

func exportProfileOperationAllowed(ctx context.Context, q *query.Queries, profileID model.ProfileID) (bool, error) {
	deletionStartedAt, err := q.GetExportProfileDeletionStartedAt(ctx, uuid.UUID(profileID))
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !deletionStartedAt.Valid, nil
}

// BeginDeletionはプロフィールの削除マーカーを永続化してから排他lockを
// 取得する。先に永続化することで後発の操作をマーカーで止め、その後のlock取得で、
// マーカー書き込み前に確認を通過したすべての操作を待つ。found=trueの場合、
// 返されたrelease関数を必ず呼ぶ。
func (r *ExportProfileDeletionGuardRepository) BeginDeletion(
	ctx context.Context,
	profileID model.ProfileID,
) (release func() error, found bool, err error) {
	if _, err := r.q.MarkExportProfileDeletionStarted(ctx, uuid.UUID(profileID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("プロフィールのexport削除開始記録に失敗: %w", err)
	}

	lock, err := r.acquire(ctx, profileID, false)
	if err != nil {
		return nil, false, err
	}
	return lock.release, true, nil
}

type exportProfileOperationLock struct {
	conn       *sql.Conn
	q          *query.Queries
	key        int64
	shared     bool
	once       sync.Once
	releaseErr error
}

func (r *ExportProfileDeletionGuardRepository) acquire(
	ctx context.Context,
	profileID model.ProfileID,
	shared bool,
) (*exportProfileOperationLock, error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("export用プロフィールlockのコネクション取得に失敗: %w", err)
	}

	key := exportProfileAdvisoryLockKey(profileID)
	queries := query.New(conn)
	if shared {
		err = queries.AcquireExportProfileOperationLock(ctx, key)
	} else {
		err = queries.AcquireExportProfileDeletionLock(ctx, key)
	}
	// 取得の成功後にもcontextを確認する。lockの取得からdriverがキャンセルに
	// 気付くまでの間にキャンセルされると、処理を続けられない呼び出し側がlockを
	// 保持したままになるため。
	//
	// いずれの場合もコネクションはプールへ返さず破棄する。取得に失敗した実行も
	// lockを取得済みでありうるため、このコネクションが保持するものを確実に解放
	// できるのはsessionを閉じることだけである。
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		discardExportProfileLockConnection(conn)
		return nil, fmt.Errorf("export用プロフィールlockの取得に失敗: %w", err)
	}

	return &exportProfileOperationLock{
		conn:   conn,
		q:      queries,
		key:    key,
		shared: shared,
	}, nil
}

func (l *exportProfileOperationLock) release() error {
	l.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), exportProfileLockReleaseTimeout)
		defer cancel()

		var unlocked bool
		if l.shared {
			unlocked, l.releaseErr = l.q.ReleaseExportProfileOperationLock(ctx, l.key)
		} else {
			unlocked, l.releaseErr = l.q.ReleaseExportProfileDeletionLock(ctx, l.key)
		}
		if l.releaseErr == nil && !unlocked {
			l.releaseErr = errors.New("保持しているexport用プロフィールlockが見つからない")
		}
		if l.releaseErr != nil {
			discardExportProfileLockConnection(l.conn)
			return
		}
		l.releaseErr = l.conn.Close()
	})
	return l.releaseErr
}

// exportProfileAdvisoryLockKeyは安定した正のPostgreSQL lock keyを導出する。
// namespace prefixで他のadvisory lock群と分離し、SHA-256のprefixから得た
// 63 bitにより衝突を無視できるほど小さくする。
func exportProfileAdvisoryLockKey(profileID model.ProfileID) int64 {
	sum := sha256.Sum256([]byte("mewst/export-profile-deletion/v1/" + profileID.String()))
	return int64(binary.BigEndian.Uint64(sum[:8]) >> 1)
}

// discardExportProfileLockConnectionは、取得または解放に失敗した後もsession
// lockを保持している可能性があるコネクションをプールへ戻さないようにする。
// driver.ErrBadConnによりdatabase/sqlが物理コネクションを破棄し、PostgreSQLは
// session終了時にそのlockをすべて解放する。
func discardExportProfileLockConnection(conn *sql.Conn) {
	_ = conn.Raw(func(any) error {
		return driver.ErrBadConn
	})
	_ = conn.Close()
}
