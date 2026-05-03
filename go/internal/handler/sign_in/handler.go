// Package sign_in はログインハンドラーを提供します
package sign_in

import (
	"net/http"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/redirect"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/templates/layouts"
	signinpages "github.com/mewstcom/mewst/go/internal/templates/pages/sign_in"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/viewmodel"
)

// Handler はログイン機能のHTTPハンドラー
type Handler struct {
	cfg               *config.Config
	sessionMgr        *session.Manager
	flashMgr          *session.FlashManager
	signInUC          *usecase.CreateSignInUsecase
	turnstileVerifier turnstile.Verifier
}

// NewHandler はHandlerを生成する
func NewHandler(
	cfg *config.Config,
	sessionMgr *session.Manager,
	flashMgr *session.FlashManager,
	signInUC *usecase.CreateSignInUsecase,
	turnstileVerifier turnstile.Verifier,
) *Handler {
	return &Handler{
		cfg:               cfg,
		sessionMgr:        sessionMgr,
		flashMgr:          flashMgr,
		signInUC:          signInUC,
		turnstileVerifier: turnstileVerifier,
	}
}

// renderSignInForm はログインフォームを描画する。
// 初回表示・バリデーションエラー時の再表示の両方から共通利用する。
// Create のバリデーション失敗時のみ呼び出し側で 422 を設定する。それ以外は status を設定せずデフォルト 200 を使う。
func (h *Handler) renderSignInForm(w http.ResponseWriter, r *http.Request, ve *model.ValidationError, email, backURL string) {
	ctx := r.Context()

	csrfToken := middleware.GetCSRFTokenFromContext(ctx)

	meta := viewmodel.DefaultPageMeta(ctx, h.cfg)
	meta.SetTitle(ctx, "sign_in_title")

	content := signinpages.New(signinpages.NewPageData{
		CSRFToken:        csrfToken,
		TurnstileSiteKey: h.cfg.TurnstileSiteKey,
		FormErrors:       ve,
		Email:            email,
		BackURL:          backURL,
		SignUpHref:       redirect.AppendSafeBack("/sign_up", backURL),
	})

	if err := layouts.Simple(layouts.SimpleLayoutData{Meta: meta}, content).Render(ctx, w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
