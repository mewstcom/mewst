package middleware

import "net/http"

// PrivateCacheControl is the Cache-Control value for a response that belongs to
// one signed-in user. "private" keeps shared caches (CDNs, proxies) from
// storing it, and "no-cache" lets the browser keep a copy but forces it to
// revalidate before reuse. "no-store" is deliberately not used: it would also
// disqualify the page from the back/forward cache, and these pages carry no
// secret beyond what the signed-in reader is already looking at.
//
// [Ja] PrivateCacheControl は 1 人のログイン中ユーザーに属する応答の
// Cache-Control の値。private は共有キャッシュ (CDN・プロキシ) への保存を防ぎ、
// no-cache はブラウザに保存を許しつつ再利用の前に必ず再検証させる。no-store を
// 使わないのは意図的で、これを付けると bfcache の対象からも外れる一方、これらの
// ページはログイン中の読み手が今見ている以上の秘密を持たないためである。
const PrivateCacheControl = "private, no-cache"

// PrivateCache marks the response as belonging to a single signed-in user.
//
// It is applied to the route groups that require authentication. Place it
// before the authentication middleware so the sign-in redirect an unauthenticated
// request receives is marked too: that redirect is served from an authenticated
// URL, and a shared cache holding it would answer a signed-in reader with it.
//
// [Ja] PrivateCache は応答が 1 人のログイン中ユーザーのものであることを示す。
//
// 認証が必要なルートグループに適用する。認証ミドルウェアより前に置くことで、
// 未認証のリクエストが受け取るログインへのリダイレクトにも付与する。そのリダイレクトは
// 認証必須の URL から返るため、共有キャッシュが保持するとログイン中の読み手へ
// それを返してしまうからである。
func PrivateCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", PrivateCacheControl)
		next.ServeHTTP(w, r)
	})
}
