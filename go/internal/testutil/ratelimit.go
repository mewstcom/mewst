package testutil

import (
	"testing"
	"time"
)

// rateLimitWindowは各ハンドラーがratelimit.CheckInputに渡す窓と同じ値。
// Limiterはtime.Now().Truncate(window) でカウントをバケット分けするため、
// この値がカウンターのリセット位置を決める。
const rateLimitWindow = time.Minute

// rateLimitTestMarginはレート制限テストが1つの窓の内側で必要とする余裕。
// 上限を使い切るテストは200msを大きく下回って終わるため、2秒あれば十分で、
// かつWaitForRateLimitWindowが加える待ち時間の平均を1呼び出しあたり
// 40ms未満に抑えられる。
const rateLimitTestMargin = 2 * time.Second

// rateLimitWindowOvershootは待ち時間に加える余剰分。境界ちょうどではなく
// 境界の直後に再開させることで、丸め次第で旧い窓に入りうる際どい位置を避ける。
const rateLimitWindowOvershoot = 10 * time.Millisecond

// WaitForRateLimitWindowは現在のレート制限の窓に
// rateLimitTestMargin以上の残りができるまで待つ。これにより、N回のリクエストで
// 上限を使い切ってからN+1回目を検証するテストの途中でカウンターがリセットされる
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
