// Package password_resetはパスワードリセット開始ハンドラーを提供します
package password_reset

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	passwordresetpages "github.com/mewstcom/mewst/go/internal/templates/pages/password_reset"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Handlerはパスワードリセット開始機能のHTTPハンドラー
type Handler struct {
	cfg                   *config.Config
	sessionMgr            *session.Manager
	flashMgr              *session.FlashManager
	createPasswordResetUC *usecase.CreatePasswordResetUsecase
	turnstileVerifier     turnstile.Verifier
}

// NewHandlerはHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	createPasswordResetUC *usecase.CreatePasswordResetUsecase,
	turnstileVerifier turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:                   cfg,
		sessionMgr:            sessionMgr,
		flashMgr:              flashMgr,
		createPasswordResetUC: createPasswordResetUC,
		turnstileVerifier:     turnstileVerifier,
	}
}

// renderPasswordResetFormはパスワードリセットフォームを描画する。
// 初回表示・バリデーションエラー時の再表示の両方から共通利用する。
// Createのバリデーション失敗時のみ呼び出し側で422を設定する。それ以外はstatusを設定せずデフォルト200を使う。
func (h *Handler) renderPasswordResetForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, email string) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "password_reset_new_title")

	content := passwordresetpages.New(passwordresetpages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       ve,
		Email:            email,
	})

	if err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
