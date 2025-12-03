package sign_out

import (
	"net/http"

	"github.com/mewstcom/mewst/internal/session"
	"github.com/mewstcom/mewst/internal/templates"
)

// Delete はログアウト処理を実行する (DELETE /sign_out)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// コンテキストにロケールを設定（翻訳用）
	ctx = templates.WithLocale(ctx, "ja")

	// セッションクッキーを削除
	h.sessionMgr.DeleteSessionCookie(w, r)

	// フラッシュメッセージを設定
	h.sessionMgr.SetFlashCookie(w, r, session.FlashSuccess, templates.T(ctx, "messages.authentication.sign_out"))

	// ホームページにリダイレクト
	http.Redirect(w, r, "/", http.StatusFound)
}
