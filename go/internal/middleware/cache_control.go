package middleware

import "net/http"

// PrivateCacheControlは1人のログイン中ユーザーに属する応答の
// Cache-Controlの値。privateは共有キャッシュ (CDN・プロキシ) への保存を防ぎ、
// no-cacheはブラウザに保存を許しつつ再利用の前に必ず再検証させる。no-storeを
// 使わないのは意図的で、これを付けるとbfcacheの対象からも外れる一方、これらの
// ページはログイン中の読み手が今見ている以上の秘密を持たないためである。
const PrivateCacheControl = "private, no-cache"

// PrivateCacheは応答が1人のログイン中ユーザーのものであることを示す。
//
// 認証が必要なルートグループに適用する。認証ミドルウェアより前に置くことで、
// 未認証のリクエストが受け取るログインへのリダイレクトにも付与する。そのリダイレクトは
// 認証必須のURLから返るため、共有キャッシュが保持するとログイン中の読み手へ
// それを返してしまうからである。
func PrivateCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", PrivateCacheControl)
		next.ServeHTTP(w, r)
	})
}
