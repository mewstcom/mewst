package sign_out

import (
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Delete signs the current user out (DELETE / POST /sign_out).
//
// [Ja] Delete はログアウト処理を実行する (DELETE / POST /sign_out)。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Delete the session row so that a copied cookie value cannot be replayed.
	// If deletion fails, a copied token may remain usable; log the error for
	// monitoring, but continue so the response still instructs the current
	// browser to clear its cookie.
	//
	// [Ja] コピー済みの Cookie 値を再利用できないよう、セッションレコードを削除する。
	// 削除に失敗するとコピー済みトークンが利用可能なまま残るため、監視できるよう
	// エラーを記録する。ただし現在のブラウザへ Cookie の削除を指示できるよう、
	// ログアウト処理は続行する。
	token := h.sessionMgr.GetSessionToken(r)
	if err := h.deleteSessionUC.Execute(ctx, usecase.DeleteSessionInput{Token: token}); err != nil {
		slog.ErrorContext(ctx, "ログアウト時のセッション削除に失敗", "error", err)
	}

	h.sessionMgr.DeleteSessionCookie(w)
	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_sign_out_success"))

	http.Redirect(w, r, "/", http.StatusFound)
}
