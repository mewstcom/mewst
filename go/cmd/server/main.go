package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/database"
	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/handler/account"
	"github.com/mewstcom/mewst/go/internal/handler/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/handler/link"
	"github.com/mewstcom/mewst/go/internal/handler/manifest"
	"github.com/mewstcom/mewst/go/internal/handler/password"
	"github.com/mewstcom/mewst/go/internal/handler/password_reset"
	"github.com/mewstcom/mewst/go/internal/handler/post"
	"github.com/mewstcom/mewst/go/internal/handler/sign_in"
	"github.com/mewstcom/mewst/go/internal/handler/sign_out"
	"github.com/mewstcom/mewst/go/internal/handler/sign_up"
	"github.com/mewstcom/mewst/go/internal/httperror"
	"github.com/mewstcom/mewst/go/internal/i18n"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/query"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
	mewstsentry "github.com/mewstcom/mewst/go/internal/sentry"
	"github.com/mewstcom/mewst/go/internal/session"
	"github.com/mewstcom/mewst/go/internal/turnstile"
	"github.com/mewstcom/mewst/go/internal/usecase"
	"github.com/mewstcom/mewst/go/internal/validator"
	"github.com/mewstcom/mewst/go/internal/worker"
)

func main() {
	// 設定を読み込み
	cfg, err := config.Load()
	if err != nil {
		slog.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}

	slog.Info("サーバーを起動します", "port", cfg.Port, "env", cfg.Env)

	// Sentry を初期化 (DSN が空の場合はスキップされる)。
	// データベース接続より前に初期化することで、起動時のエラーも Sentry に送信できる。
	if err := mewstsentry.Init(mewstsentry.Config{
		DSN:              cfg.SentryDSN,
		Environment:      cfg.SentryEnvironment,
		Release:          cfg.AssetVersion,
		TracesSampleRate: cfg.SentryTracesSampleRate,
		Debug:            cfg.SentryDebug,
	}); err != nil {
		slog.Error("Sentry の初期化に失敗しました", "error", err)
		os.Exit(1)
	}
	defer mewstsentry.Flush(2 * time.Second)

	// slog のデフォルトハンドラーを「標準出力 + Sentry」のファンアウトに差し替える。
	// これ以降の slog.ErrorContext / slog.Error 呼び出しは自動的に Sentry のイベントとして送信される。
	// 各層 (handler / usecase / validator / repository / middleware) で sentry.CaptureError を
	// 明示的に呼ぶ必要がない。
	// Sentry 初期化失敗時のログは引き続き Go 1.21+ のデフォルトハンドラー (TextHandler @ stderr) に出すべきため、
	// SetDefault は mewstsentry.Init 成功後に呼ぶ。
	baseSlogHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(mewstsentry.NewSlogHandler(baseSlogHandler)))

	// データベース接続
	db, err := database.Connect(cfg.DatabaseDSN())
	if err != nil {
		slog.Error("データベース接続に失敗しました", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("データベース接続のクローズに失敗しました", "error", err)
		}
	}()
	slog.Info("データベースに接続しました")

	// クエリの初期化
	queries := query.New(db)

	// リポジトリの初期化
	userRepo := repository.NewUserRepository(queries)
	sessionRepo := repository.NewSessionRepository(queries)
	actorRepo := repository.NewActorRepository(queries)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(queries)
	profileRepo := repository.NewProfileRepository(queries)
	userProfileRepo := repository.NewUserProfileRepository(queries)
	rateLimitRepo := repository.NewRateLimitRepository(queries)
	featureFlagRepo := repository.NewFeatureFlagRepository(queries)
	postRepo := repository.NewPostRepository(queries)
	followRepo := repository.NewFollowRepository(queries)
	homeTimelinePostRepo := repository.NewHomeTimelinePostRepository(queries)
	oauthApplicationRepo := repository.NewOauthApplicationRepository(queries)
	linkRepo := repository.NewLinkRepository(queries)
	postLinkRepo := repository.NewPostLinkRepository(queries)

	// セッションマネージャーの初期化
	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)
	flashMgr := session.NewFlashManager(cfg.CookieDomain, cfg.SessionSecure, cfg.SessionHTTPOnly)

	// Build the Dispatcher first, backed by a DeferredInserter. FanoutPostUsecase
	// needs the Dispatcher, the Dispatcher needs the River client, and the River
	// client can only be built after worker.NewClient registers the Workers that
	// wrap those UseCases — an initialization cycle. Construct the Dispatcher
	// around an unwired DeferredInserter now and inject the River client via
	// SetInserter once worker.NewClient has created it.
	//
	// [Ja] FanoutPostUsecase → Dispatcher → River クライアント → Worker (UseCase を内包) の
	// 初期化循環を断つため、先に DeferredInserter で Dispatcher を構築し、River クライアント
	// 生成後に注入する。
	deferredInserter := &dispatcher.DeferredInserter{}
	jobDispatcher := dispatcher.NewDispatcher(deferredInserter)

	// Build the fanout UseCases that the Workers register. They depend on
	// repository, so they cannot be built inside the worker package (depguard
	// forbids worker → repository); build them here and inject them.
	// [Ja] Worker に登録する fanout 系 UseCase を構築する (repository 依存のため worker 内で
	// は構築できず、ここで注入する)。
	fanoutPostUC := usecase.NewFanoutPostUsecase(postRepo, followRepo, jobDispatcher)
	addPostToTimelineUC := usecase.NewAddPostToTimelineUsecase(profileRepo, postRepo, homeTimelinePostRepo)

	// Workerの初期化
	workerClient, err := worker.NewClient(context.Background(), cfg.DatabaseDSN(), cfg, fanoutPostUC, addPostToTimelineUC)
	if err != nil {
		slog.Error("Workerクライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}

	// Wire the real inserter into the DeferredInserter now that the River client
	// exists (completes the wiring that broke the init cycle).
	// [Ja] River クライアント生成後に DeferredInserter へ実体を注入する (循環を断った配線の完了)。
	deferredInserter.SetInserter(workerClient.Client())

	// Workerを開始
	if err := workerClient.Start(context.Background()); err != nil {
		slog.Error("Workerの開始に失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("Workerを開始しました")

	// レートリミッターの初期化
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	// バリデーターの初期化
	signInValidator := validator.NewSignInCreateValidator(userRepo)
	signUpValidator := validator.NewSignUpCreateValidator(userRepo)
	emailConfirmationValidator := validator.NewEmailConfirmationCreateValidator(emailConfirmationRepo)
	accountValidator := validator.NewAccountCreateValidator(userRepo, profileRepo)
	passwordUpdateValidator := validator.NewPasswordUpdateValidator()
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()
	postCreateValidator := validator.NewPostCreateValidator()
	linkDataFetcherValidator := validator.NewLinkDataFetcherValidator()

	// ユースケースの初期化
	createSessionUC := usecase.NewCreateSessionUsecase(actorRepo, sessionRepo)
	createSignInUC := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)
	createSignUpUC := usecase.NewCreateSignUpUsecase(signUpValidator, emailConfirmationRepo, jobDispatcher)
	createPasswordResetUC := usecase.NewCreatePasswordResetUsecase(passwordResetCreateValidator, emailConfirmationRepo, jobDispatcher)
	verifyEmailConfirmationUC := usecase.NewVerifyEmailConfirmationUsecase(emailConfirmationValidator, emailConfirmationRepo)
	getActiveEmailConfirmationUC := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmationRepo)
	getSucceededEmailConfirmationUC := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmationRepo)
	updatePasswordUC := usecase.NewUpdatePasswordUsecase(passwordUpdateValidator, userRepo)
	createAccountUC := usecase.NewCreateAccountUsecase(db, accountValidator, userRepo, profileRepo, userProfileRepo, actorRepo)
	createPostUC := usecase.NewCreatePostUsecase(db, postCreateValidator, oauthApplicationRepo, linkRepo, postRepo, postLinkRepo, profileRepo, homeTimelinePostRepo, jobDispatcher)
	// blockPrivateHosts is true in production wiring: fetching user-supplied URLs
	// must not reach internal hosts (SSRF). Passing false still compiles and
	// passes tests, so review wiring changes here carefully. The 10s timeout
	// bounds each fetch (applied per redirect hop) so a slow external site cannot
	// pin a request handler indefinitely.
	//
	// [Ja] 本番配線では blockPrivateHosts を true にする。ユーザー入力の URL の
	// 取得が内部ホストへ到達してはならない (SSRF 対策)。false を渡しても
	// コンパイル・テストは通るため、この配線の変更は注意して見ること。10 秒の
	// タイムアウトは取得 1 回ごと (リダイレクトの各ホップ) に適用され、遅い外部
	// サイトがリクエストハンドラーを占有し続けないようにする。
	fetchLinkMetadataUC := usecase.NewFetchLinkMetadataUsecase(linkDataFetcherValidator, linkRepo, &http.Client{Timeout: 10 * time.Second}, true)

	// Turnstileクライアントの初期化
	turnstileClient := turnstile.NewClient(cfg.TurnstileSecretKey)

	// ハンドラーの初期化
	manifestHandler := manifest.NewHandler(cfg)
	signInHandler := sign_in.NewHandler(cfg, sessionMgr, flashMgr, createSignInUC, turnstileClient)
	signUpHandler := sign_up.NewHandler(cfg, sessionMgr, flashMgr, createSignUpUC, turnstileClient, rateLimiter)
	signOutHandler := sign_out.NewHandler(cfg, sessionMgr, flashMgr)
	passwordResetHandler := password_reset.NewHandler(cfg, sessionMgr, flashMgr, createPasswordResetUC, turnstileClient)
	emailConfirmationHandler := email_confirmation.NewHandler(cfg, sessionMgr, flashMgr, getActiveEmailConfirmationUC, verifyEmailConfirmationUC)
	passwordHandler := password.NewHandler(cfg, sessionMgr, flashMgr, getSucceededEmailConfirmationUC, updatePasswordUC)
	accountHandler := account.NewHandler(cfg, sessionMgr, flashMgr, getSucceededEmailConfirmationUC, createAccountUC, createSessionUC, turnstileClient, rateLimiter)
	postHandler := post.NewHandler(cfg, flashMgr, createPostUC)
	linkHandler := link.NewHandler(fetchLinkMetadataUC)

	// ミドルウェアの初期化
	authMiddleware := middleware.NewAuth(sessionMgr)
	csrfMiddleware := middleware.NewCSRF(cfg)
	sentryUserContextMW := middleware.NewSentryUserContext(profileRepo)

	// Sentry の HTTP ミドルウェア。
	// Repanic: true により、panic を Sentry に送ったあと再 panic させて後続の Recoverer に処理を委ねる。
	sentryHTTPHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})

	// Chiルーターの設定
	r := chi.NewRouter()

	// 基本ミドルウェア
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestID)
	// Client IP is resolved on demand via internal/clientip (CF-Connecting-IP first),
	// so chi's IP middleware is intentionally not registered. chi's RealIP is also
	// deprecated for IP spoofing (GHSA-3fxj-6jh8-hvhx) and must not be reintroduced.
	//
	// [Ja] クライアント IP は internal/clientip (CF-Connecting-IP 優先) で都度解決するため、
	// chi の IP ミドルウェアは意図的に登録しない。chi の RealIP は IP spoofing
	// (GHSA-3fxj-6jh8-hvhx) のため deprecated でもあり、再導入しないこと。

	// Recoverer を outer (chi の Use では最初に登録 = チェーンの一番外側) に置く。
	// chi の Recoverer は http.ErrAbortHandler 以外を recover して 500 を書き、再 panic しないため、
	// sentryhttp より「外側」(= panic 伝搬の最後に届く位置) に置かないと、sentryhttp が panic を見られない。
	r.Use(chimiddleware.Recoverer)
	// sentryhttp は Recoverer より「あとに登録」= chi のチェーンでは innermost (= handler に近い側)。
	// handler の panic を sentryhttp の defer がまず捕捉し、Sentry に送信したあと Repanic: true で再 panic。
	// 再 panic は outer の Recoverer に到達し、Recoverer が 500 レスポンスを書いて終了する。
	r.Use(sentryHTTPHandler.Handle)
	// chi のルートパターン (例: "/users/{id}") を Sentry のトランザクション名に上書きする。
	// sentryhttp より「あとに登録」することで、defer (LIFO) のタイミングで sentryhttp の
	// transaction.Finish() より先に Name を確定できる。
	r.Use(middleware.SentryTransaction)

	// リバースプロキシの設定（Rails版へのプロキシ）
	if cfg.RailsAppURL != "" {
		proxyMiddleware, err := middleware.NewReverseProxyMiddleware(cfg.RailsAppURL, cfg, featureFlagRepo)
		if err != nil {
			slog.Error("リバースプロキシミドルウェアの初期化に失敗しました", "error", err)
			os.Exit(1)
		}
		r.Use(proxyMiddleware.Middleware)
	}

	// Limit the request body size for Go-handled routes. Placed after reverse_proxy so
	// requests proxied to the Rails version are not affected, and before any middleware /
	// handler that parses form data (CSRF, MethodOverride, handlers).
	//
	// [Ja] Go 版が処理するルートのリクエストボディサイズを制限する。reverse_proxy より
	// 後に配置することで Rails 版にプロキシされるリクエストには影響させず、フォームを
	// パースするミドルウェア / ハンドラー (CSRF、MethodOverride、各ハンドラー) より前に置く。
	r.Use(middleware.BodyLimit)

	// i18n ミドルウェア(Accept-Language ヘッダーから ctx にロケールをセットする)
	// reverse_proxy より後に配置することで、Rails 版にプロキシされるリクエストには走らせない
	r.Use(i18n.Middleware)

	// フラッシュメッセージをCookieからcontextへロード（Go版の全ルートに適用）
	r.Use(flashMgr.Middleware)

	// 404ハンドラーの設定（ルーティングにマッチしないパス用）
	r.NotFound(httperror.NotFound)

	// 静的ファイルの配信 (Tailwind CLI + esbuild のビルド結果)
	fileServer := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// ヘルスチェックエンドポイント
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// DBの接続確認
		if err := db.PingContext(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "ヘルスチェック: データベース接続エラー", "error", err)
			http.Error(w, "DB connection failed", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Web App Manifest
	r.Get("/manifest.json", manifestHandler.Show)

	// ログイン・サインアップページ（未認証ユーザーのみ）
	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireNoAuth)
		r.Use(sentryUserContextMW.Middleware)
		r.Get("/sign_in", signInHandler.New)
		r.Post("/sign_in", signInHandler.Create)
		r.Get("/sign_up", signUpHandler.New)
		r.Post("/sign_up", signUpHandler.Create)
	})

	// ログアウト（認証済みユーザーのみ）
	// HTMLフォームからはPOST（_method=DELETE付き）で呼び出されるが、
	// Chiはルートマッチング時にメソッドも考慮するため、
	// POSTとDELETE両方を登録する必要がある
	// 注: Rails版のページからのリクエストにはGo版のCSRFトークンが含まれないため、
	// CSRFミドルウェアは適用しない（ログアウトは破壊的操作ではないため安全）
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Use(sentryUserContextMW.Middleware)
		r.Delete("/sign_out", signOutHandler.Delete)
		r.Post("/sign_out", signOutHandler.Delete)
	})

	// パスワードリセット・メール確認・アカウント作成（未認証ユーザーのみ）
	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireNoAuth)
		r.Use(sentryUserContextMW.Middleware)

		// パスワードリセット開始（メールアドレス入力）
		r.Get("/password_reset", passwordResetHandler.New)
		r.Post("/password_reset", passwordResetHandler.Create)

		// メール確認（確認コード入力）
		r.Get("/email_confirmation", emailConfirmationHandler.New)
		r.Post("/email_confirmation", emailConfirmationHandler.Create)

		// パスワード更新（新しいパスワード設定）
		r.Get("/password/edit", passwordHandler.Edit)
		r.Patch("/password", passwordHandler.Update)
		r.Post("/password", passwordHandler.Update) // HTMLフォームからのPOST対応（_method=PATCH）

		// アカウント作成（メール確認後）
		r.Get("/accounts/new", accountHandler.New)
		r.Post("/accounts", accountHandler.Create)
	})

	// New post form, submission, and link card fragments (authenticated users
	// only). All routes are gated by the reverse-proxy feature flag
	// (FeatureFlagNewPost), so requests from viewers without the flag never reach
	// them and are proxied to the Rails version. POST /posts and POST /links are
	// genuine POSTs, so no Method Override is needed; the routes are registered
	// directly. GET /links/new sits under the CSRF middleware so the fragment can
	// embed the CSRF token for its POST /links form.
	//
	// [Ja] 新規投稿フォーム・送信・リンクカードのフラグメント（認証済みユーザーのみ）。
	// すべてのルートはリバースプロキシのフィーチャーフラグ (FeatureFlagNewPost) で
	// ゲートされるため、フラグが無効な閲覧者のリクエストはここに到達せず Rails 版に
	// プロキシされる。POST /posts と POST /links は本来の POST のため Method Override
	// は不要で、ルートを直接登録する。GET /links/new はフラグメントが POST /links
	// フォーム用の CSRF トークンを埋め込めるよう CSRF ミドルウェア配下に置く。
	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireAuth)
		r.Use(sentryUserContextMW.Middleware)

		r.Get("/new", postHandler.New)
		r.Post("/posts", postHandler.Create)
		r.Get("/links/new", linkHandler.New)
		r.Post("/links", linkHandler.Create)
	})

	// サーバー起動
	// Dockerコンテナ内で動かす場合、0.0.0.0でリッスンする必要がある
	addr := fmt.Sprintf("0.0.0.0:%s", cfg.Port)
	slog.Info("HTTPサーバーを起動します", "addr", addr)

	// HTTPサーバーの作成
	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Graceful shutdown のためのシグナルハンドリング
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		slog.Info("シャットダウンシグナルを受信しました。サーバーを停止します...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Workerを停止
		if err := workerClient.Stop(shutdownCtx); err != nil {
			slog.Error("Workerの停止に失敗しました", "error", err)
		} else {
			slog.Info("Workerを停止しました")
		}

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("サーバーのシャットダウンに失敗しました", "error", err)
		}
	}()

	// サーバー起動
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("サーバーの起動に失敗しました", "error", err)
		os.Exit(1)
	}

	slog.Info("サーバーが正常に停止しました")
}
