package testutil

import (
	"testing"
	"time"
)

// rateLimitWindow mirrors the window every handler passes to
// ratelimit.CheckInput. The limiter buckets counts by time.Now().Truncate(window),
// so this value determines where the counter resets.
//
// [Ja] rateLimitWindow は各ハンドラーが ratelimit.CheckInput に渡す窓と同じ値。
// Limiter は time.Now().Truncate(window) でカウントをバケット分けするため、
// この値がカウンターのリセット位置を決める。
const rateLimitWindow = time.Minute

// rateLimitTestMargin is the headroom a rate limit test needs inside one window.
// The tests that exhaust a limit finish in well under 200ms, so 2s leaves ample
// room while keeping the average wait added by WaitForRateLimitWindow under
// 40ms per call.
//
// [Ja] rateLimitTestMargin はレート制限テストが 1 つの窓の内側で必要とする余裕。
// 上限を使い切るテストは 200ms を大きく下回って終わるため、2 秒あれば十分で、
// かつ WaitForRateLimitWindow が加える待ち時間の平均を 1 呼び出しあたり
// 40ms 未満に抑えられる。
const rateLimitTestMargin = 2 * time.Second

// rateLimitWindowOvershoot is added to the wait so execution resumes just after
// the boundary rather than exactly on it, which keeps the test off the edge
// where rounding could still place it in the old window.
//
// [Ja] rateLimitWindowOvershoot は待ち時間に加える余剰分。境界ちょうどではなく
// 境界の直後に再開させることで、丸め次第で旧い窓に入りうる際どい位置を避ける。
const rateLimitWindowOvershoot = 10 * time.Millisecond

// WaitForRateLimitWindow blocks until the current rate limit window has at
// least rateLimitTestMargin left, so that a test which sends N requests to
// exhaust a limit and then asserts on request N+1 cannot have the counter reset
// underneath it.
//
// Rate limits use a fixed window keyed on wall-clock time, so a test straddling
// a window boundary sees its counter go back to zero and the request that
// should have been rejected succeeds instead. Call this immediately before
// sending the first request of such a test.
//
// [Ja] WaitForRateLimitWindow は現在のレート制限の窓に
// rateLimitTestMargin 以上の残りができるまで待つ。これにより、N 回のリクエストで
// 上限を使い切ってから N+1 回目を検証するテストの途中でカウンターがリセットされる
// ことを防ぐ。
//
// レート制限はウォールクロック基準の固定窓を使うため、窓の境界をまたいだテストでは
// カウンターがゼロに戻り、本来拒否されるはずのリクエストが通ってしまう。
// この種のテストで最初のリクエストを送る直前に呼び出すこと。
func WaitForRateLimitWindow(t *testing.T) {
	t.Helper()

	now := time.Now().UTC()
	remaining := rateLimitWindow - now.Sub(now.Truncate(rateLimitWindow))
	if remaining >= rateLimitTestMargin {
		return
	}

	time.Sleep(remaining + rateLimitWindowOvershoot)
}
