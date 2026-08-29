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

// exportProfileLockReleaseTimeout bounds release on a context detached from
// the operation's context. A canceled upload still has to release its
// session-scoped lock before the connection returns to the pool.
//
// [Ja] exportProfileLockReleaseTimeout は、操作の context から切り離した lock 解放の
// 上限時間。キャンセルされた upload でも、session 単位の lock をコネクションプールへ
// 返す前に解放する必要がある。
const exportProfileLockReleaseTimeout = 5 * time.Second

// ExportProfileDeletionGuardRepository coordinates export operations with
// profile deletion. A persistent marker closes the profile to new work, while
// PostgreSQL advisory locks let deletion wait for work that already crossed
// the boundary without holding a regular transaction across external I/O.
//
// [Ja] ExportProfileDeletionGuardRepository は export 操作とプロフィール削除を
// 調整する。永続マーカーで新しい処理を閉じ、PostgreSQL advisory lock により、外部
// I/O の間に通常 transaction を保持せず、削除が境界を越えた処理を待てるようにする。
type ExportProfileDeletionGuardRepository struct {
	db *sql.DB
	q  *query.Queries
}

// NewExportProfileDeletionGuardRepository creates an
// ExportProfileDeletionGuardRepository.
//
// [Ja] NewExportProfileDeletionGuardRepository は
// ExportProfileDeletionGuardRepository を生成する。
func NewExportProfileDeletionGuardRepository(db *sql.DB) *ExportProfileDeletionGuardRepository {
	return &ExportProfileDeletionGuardRepository{
		db: db,
		q:  query.New(db),
	}
}

// BeginOperation checks the persistent deletion marker before waiting for the
// profile's shared operation lock, then checks it again while holding that
// lock. The first check makes work submitted after deletion stop immediately
// instead of waiting behind deletion's queued exclusive lock. The second closes
// the race where deletion starts between the first check and lock acquisition.
// The returned release function must be called when allowed is true. A missing
// or deleting profile returns allowed=false.
//
// [Ja] BeginOperation は永続的な削除マーカーを確認してからプロフィールの操作用
// 共有 lock を待ち、取得後にもう一度確認する。最初の確認により、削除開始後に投入
// された処理は、削除用の排他 lock の後ろで待たず即座に停止する。2 回目の確認は、
// 最初の確認と lock 取得の間に削除が始まる競合を閉じる。allowed=true の場合、
// 返された release 関数を必ず呼ぶ。存在しない、または削除中のプロフィールでは
// allowed=false を返す。
func (r *ExportProfileDeletionGuardRepository) BeginOperation(
	ctx context.Context,
	profileID model.ProfileID,
) (release func() error, allowed bool, err error) {
	allowed, err = exportProfileOperationAllowed(ctx, r.q, profileID)
	if err != nil {
		return nil, false, fmt.Errorf("プロフィールの export 操作可否の事前確認に失敗: %w", err)
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
			fmt.Errorf("プロフィールの export 操作可否の確認に失敗: %w", checkErr),
			releaseErr,
		)
	}
	if !allowed {
		if err := lock.release(); err != nil {
			return nil, false, fmt.Errorf("export 操作用プロフィール lock の解放に失敗: %w", err)
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

// BeginDeletion persists the profile's deletion marker and then takes its
// exclusive lock. Persisting first makes later operations stop at the marker;
// taking the lock afterward waits for every operation that passed the check
// before the marker was written. The returned release function must be called
// when found is true.
//
// [Ja] BeginDeletion はプロフィールの削除マーカーを永続化してから排他 lock を
// 取得する。先に永続化することで後発の操作をマーカーで止め、その後の lock 取得で、
// マーカー書き込み前に確認を通過したすべての操作を待つ。found=true の場合、
// 返された release 関数を必ず呼ぶ。
func (r *ExportProfileDeletionGuardRepository) BeginDeletion(
	ctx context.Context,
	profileID model.ProfileID,
) (release func() error, found bool, err error) {
	if _, err := r.q.MarkExportProfileDeletionStarted(ctx, uuid.UUID(profileID)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("プロフィールの export 削除開始記録に失敗: %w", err)
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
		return nil, fmt.Errorf("export 用プロフィール lock のコネクション取得に失敗: %w", err)
	}

	key := exportProfileAdvisoryLockKey(profileID)
	queries := query.New(conn)
	if shared {
		err = queries.AcquireExportProfileOperationLock(ctx, key)
	} else {
		err = queries.AcquireExportProfileDeletionLock(ctx, key)
	}
	// The context is checked again after a successful acquisition: it can be
	// cancelled between the lock being taken and the driver noticing, which
	// leaves a caller that is no longer allowed to proceed holding the lock.
	//
	// Either way the connection is discarded rather than pooled. A failed
	// acquisition may still have taken the lock, and only closing the session
	// releases what this connection holds for certain.
	//
	// [Ja] 取得の成功後にも context を確認する。lock の取得から driver がキャンセルに
	// 気付くまでの間にキャンセルされると、処理を続けられない呼び出し側が lock を
	// 保持したままになるため。
	//
	// いずれの場合もコネクションはプールへ返さず破棄する。取得に失敗した実行も
	// lock を取得済みでありうるため、このコネクションが保持するものを確実に解放
	// できるのは session を閉じることだけである。
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		discardExportProfileLockConnection(conn)
		return nil, fmt.Errorf("export 用プロフィール lock の取得に失敗: %w", err)
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
			l.releaseErr = errors.New("保持している export 用プロフィール lock が見つからない")
		}
		if l.releaseErr != nil {
			discardExportProfileLockConnection(l.conn)
			return
		}
		l.releaseErr = l.conn.Close()
	})
	return l.releaseErr
}

// exportProfileAdvisoryLockKey derives a stable positive PostgreSQL lock key.
// A namespace prefix keeps this lock family independent from other advisory
// locks, and 63 bits from a SHA-256 prefix make collisions negligible.
//
// [Ja] exportProfileAdvisoryLockKey は安定した正の PostgreSQL lock key を導出する。
// namespace prefix で他の advisory lock 群と分離し、SHA-256 の prefix から得た
// 63 bit により衝突を無視できるほど小さくする。
func exportProfileAdvisoryLockKey(profileID model.ProfileID) int64 {
	sum := sha256.Sum256([]byte("mewst/export-profile-deletion/v1/" + profileID.String()))
	return int64(binary.BigEndian.Uint64(sum[:8]) >> 1)
}

// discardExportProfileLockConnection prevents a connection that may still hold
// a session lock after acquisition or release failed from returning to the
// pool. driver.ErrBadConn tells database/sql to discard the physical
// connection, and PostgreSQL releases all of that session's locks when it
// closes.
//
// [Ja] discardExportProfileLockConnection は、取得または解放に失敗した後も session
// lock を保持している可能性があるコネクションをプールへ戻さないようにする。
// driver.ErrBadConn により database/sql が物理コネクションを破棄し、PostgreSQL は
// session 終了時にその lock をすべて解放する。
func discardExportProfileLockConnection(conn *sql.Conn) {
	_ = conn.Raw(func(any) error {
		return driver.ErrBadConn
	})
	_ = conn.Close()
}
