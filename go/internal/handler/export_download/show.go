package export_download

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/model"
	"github.com/mewstcom/mewst/go/internal/usecase"
)

// archiveContentType is the media type of an export archive. It is set
// explicitly so the response never depends on net/http sniffing the first
// bytes of a stream the server did not produce itself.
//
// [Ja] archiveContentType はエクスポートのアーカイブのメディアタイプ。明示的に
// 設定することで、サーバー自身が生成したのではないストリームの先頭バイトを
// net/http が判定した結果にレスポンスが依存しないようにする。
const archiveContentType = "application/zip"

// archiveCacheControl keeps the archive out of every cache. The route group
// marks its pages "private, no-cache", which lets the browser keep a copy and
// revalidate it; the archive goes further because it is a file the reader
// downloads once and a new export replaces it in place, so a stored copy would
// be served for an archive that no longer exists.
//
// [Ja] archiveCacheControl はアーカイブをあらゆるキャッシュから外す。ルート
// グループはページを "private, no-cache" とし、ブラウザに複製の保持と再検証を
// 許すが、アーカイブはさらに踏み込む。読み手が一度ダウンロードするファイルであり、
// 新しいエクスポートが同じ場所でそれを置き換えるため、保存された複製は既に存在
// しないアーカイブとして返されることになるからである。
const archiveCacheControl = "private, no-store"

// Show hands over the profile's downloadable export archive
// (GET /settings/export/download).
//
// The signed-in user and profile are the whole request: which archive exists
// and whether this reader may have it is decided by the UseCase from that pair,
// so this handler never compares IDs or names an object itself. What it adds is
// the description of the response — the media type, the file name a file
// manager shows, and the caching and sniffing rules a downloaded file needs —
// and the transfer of the stream.
//
// [Ja] Show はプロフィールのダウンロード可能なエクスポートのアーカイブを渡す
// (GET /settings/export/download)。
//
// リクエストの内容はログイン中のユーザーとプロフィールがすべてである。どの
// アーカイブが存在し、この読み手がそれを得てよいかは UseCase がその組から判断する
// ため、このハンドラーは ID を比較することも、オブジェクトを自ら名指しすることも
// しない。ここで加えるのはレスポンスの説明 (メディアタイプ、ファイルアプリが表示
// するファイル名、ダウンロードされるファイルに必要なキャッシュと sniffing の規則)
// とストリームの転送である。
func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// RequireAuth puts both on the context before this handler runs, so a
	// missing one means the route was wired without it rather than a signed-out
	// visitor. Fail with 500 instead of opening an archive for an unknown owner.
	//
	// [Ja] RequireAuth がこのハンドラーの前に両方を context へ格納するため、欠けて
	// いる場合は未ログインではなく RequireAuth 無しでルートを登録したことを意味する。
	// 所有者が不明なままアーカイブを開かず、500 で失敗させる。
	user := middleware.UserFromContext(ctx)
	profile := middleware.ProfileFromContext(ctx)
	if user == nil || profile == nil {
		slog.ErrorContext(ctx, "エクスポートのダウンロードでログイン中のユーザーまたはプロフィールを取得できませんでした")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	output, err := h.getExportDownloadUC.Execute(ctx, usecase.GetExportDownloadInput{
		UserID:    user.ID,
		ProfileID: profile.ID,
	})
	if err != nil {
		handleShowError(w, r, err)
		return
	}
	defer func() {
		if err := output.Body.Close(); err != nil {
			slog.WarnContext(ctx, "エクスポートのストリームのクローズに失敗", "error", err)
		}
	}()

	// The file name is built by the UseCase from a fixed format and a date, so
	// it holds nothing that needs the extended encoding of RFC 6266. Quoting it
	// is enough to keep it one token to the client.
	//
	// [Ja] ファイル名は UseCase が固定の書式と日付から組み立てるため、RFC 6266 の
	// 拡張エンコーディングを要するものを含まない。クライアントへ 1 つのトークンとして
	// 渡すには引用符で囲めば足りる。
	w.Header().Set("Content-Type", archiveContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", output.FileName))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", archiveCacheControl)
	// The size is known before the first byte is sent, so it is declared rather
	// than left to chunked transfer. A client that knows the length can show
	// progress on a download that takes a while, and can tell a complete
	// archive from a connection that ended early.
	//
	// [Ja] サイズは最初のバイトを送る前に判明しているため、chunked 転送に委ねず
	// 宣言する。長さが分かるクライアントは、時間のかかるダウンロードの進捗を表示でき、
	// 完全なアーカイブと途中で終わった接続を区別できる。
	w.Header().Set("Content-Length", strconv.FormatInt(output.Size, 10))
	w.WriteHeader(http.StatusOK)

	// The bytes are copied as they are: the archive is already compressed, and
	// the response declares its length, so nothing here re-encodes it.
	//
	// A failed copy means the response body could not be delivered — the reader
	// cancelled the download or the connection dropped — or that the object
	// stream ended early. Neither is something the server can answer
	// differently once the header is out, and the first is an ordinary way for
	// a download to end, so it is logged at warn: Sentry captures error and
	// above, and this belongs in the local logs rather than in an alert.
	//
	// [Ja] バイト列はそのまま転送する。アーカイブは既に圧縮されており、レスポンスは
	// 長さを宣言しているため、ここで再エンコードするものは無い。
	//
	// 転送の失敗は、レスポンスボディを送り切れなかったこと (読み手がダウンロードを
	// 中断した、接続が切れた) か、オブジェクトのストリームが途中で終わったことを
	// 意味する。どちらもヘッダー送出後にサーバーが別の応答を返せるものではなく、
	// 前者はダウンロードの終わり方として通常のものであるため warn で記録する。
	// Sentry は error 以上を送るため、これはアラートではなくローカルのログに残る。
	if _, err := io.Copy(w, output.Body); err != nil {
		slog.WarnContext(ctx, "エクスポートのアーカイブの転送に失敗", "error", err)
	}
}

// handleShowError turns a refused or failed download into a response.
//
// Nothing has been written to w at this point, so each outcome still chooses
// its own status and body. A profile the user does not own and a profile with
// no archive are both answered as not found by the UseCase, which is what keeps
// the response from telling the two apart.
//
// [Ja] handleShowError は拒否または失敗したダウンロードをレスポンスに変換する。
//
// この時点で w には何も書き込んでいないため、どの結果もまだ自身の status と本文を
// 選べる。所有していないプロフィールと、アーカイブを持たないプロフィールは、
// どちらも UseCase が not found として答える。これが応答から両者を区別できないように
// している。
func handleShowError(w http.ResponseWriter, r *http.Request, err error) {
	ctx := r.Context()

	var ae *model.AppError
	if errors.As(err, &ae) {
		switch ae.Code {
		case model.AppErrCodeResourceNotFound:
			httperror.NotFound(w, r)
		case model.AppErrCodeServiceUnavailable:
			// The download link is not rendered on a deployment that cannot
			// serve archives, so reaching here means the request did not come
			// from the page as it stands. Answer with the status that says the
			// feature is not being served rather than with a page describing it.
			//
			// [Ja] アーカイブを提供できないデプロイではダウンロードリンクを描画
			// しないため、ここへ到達したリクエストは現在の画面から来たものではない。
			// 機能を説明するページではなく、提供していないことを表す status で答える。
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		default:
			slog.ErrorContext(ctx, ae.LogString())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	slog.ErrorContext(ctx, "エクスポートのダウンロードに失敗", "error", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
