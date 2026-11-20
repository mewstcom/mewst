package export

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/templates"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// Createはログイン中プロフィールのエクスポートを開始する
// (POST /settings/export)。フォームはミドルウェアが検証するCSRFトークン以外の
// フィールドを持たないため、リクエストの内容はcontext上のidentityがすべてである。
// ユーザーとプロフィールが「誰のポストをエクスポートするか」と「それが許されるか」を
// 決め、actorは申請者として記録される。
//
// どの結果もエクスポート画面へ着く。このリクエストが生んだ状態を説明するのはその
// 画面であるため。成功と拒否で異なるのは残すflashであって、読み手の行き先ではない。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// RequireAuthがこのハンドラーの前に3つともcontextへ格納するため、
	// 欠けている場合は未ログインではなくRequireAuth無しでルートを登録したことを
	// 意味する。所有者が不明なままエクスポートを作らず、500で失敗させる。
	user := middleware.UserFromContext(ctx)
	profile := middleware.ProfileFromContext(ctx)
	actor := middleware.ActorFromContext(ctx)
	if user == nil || profile == nil || actor == nil {
		slog.ErrorContext(ctx, "エクスポート開始でログイン中のユーザー・プロフィール・アクターを取得できませんでした")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if _, err := h.createExportUC.Execute(ctx, usecase.CreateExportInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
		ActorID:   actor.ID,
	}); err != nil {
		h.handleCreateError(w, r, err)
		return
	}

	h.flashMgr.SetSuccess(w, i18n.T(ctx, "flash_export_started"))
	http.Redirect(w, r, string(templates.SettingExportPath()), http.StatusFound)
}

// handleCreateErrorはエクスポート開始の失敗をレスポンスに変換する。
//
// 競合はプロフィール自身の状態による拒否 (既にエクスポートが実行中、あるいは
// プロフィールが削除処理中) であるため、その状態を説明する画面へ戻し、UseCaseの
// メッセージを警告として見せる。メッセージをここで選んだキーではなくエラーから
// 取るのは、どの競合が起きたかがUseCaseの判断であり、2つの競合はHTTP
// レスポンスに同じ痕跡しか残さないためである。
func (h *Handler) handleCreateError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	var ae *model.AppError
	if errors.As(err, &ae) {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			httperror.NotFound(w, r)
		case model.AppErrCodeConflict:
			slog.InfoContext(ctx, ae.LogString())
			h.flashMgr.SetWarning(w, ae.UserMsg)
			http.Redirect(w, r, string(templates.SettingExportPath()), http.StatusFound)
		case model.AppErrCodeServiceUnavailable:
			// エクスポートを実行できないデプロイでは開始ボタンを描画しないため、
			// ここへ到達したリクエストは現在の画面から来たものではない。機能を説明する
			// ページではなく、提供していないことを表すstatusで答える。
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "エクスポートの開始に失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
