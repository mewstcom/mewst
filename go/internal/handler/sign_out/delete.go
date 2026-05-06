package sign_out

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
)

// Delete はログアウト処理を実行する (DELETE /sign_out)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// セッションクッキーを削除
	h.sessionMgr.DeleteSessionCookie(w)

	// フラッシュメッセージを設定
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_sign_out_success"))

	// ホームページにリダイレクト
	http.Redirect(w, r, "/", http.StatusFound)
}
