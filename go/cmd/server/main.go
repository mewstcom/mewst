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

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/mewstcom/mewst/go/internal/config"
	"github.com/mewstcom/mewst/go/internal/database"
	"github.com/mewstcom/mewst/go/internal/dispatcher"
	"github.com/mewstcom/mewst/go/internal/email"
	"github.com/mewstcom/mewst/go/internal/handler"
	"github.com/mewstcom/mewst/go/internal/handler/accounts"
	"github.com/mewstcom/mewst/go/internal/handler/email_confirmation"
	"github.com/mewstcom/mewst/go/internal/handler/manifest"
	"github.com/mewstcom/mewst/go/internal/handler/password"
	"github.com/mewstcom/mewst/go/internal/handler/password_reset"
	"github.com/mewstcom/mewst/go/internal/handler/sign_in"
	"github.com/mewstcom/mewst/go/internal/handler/sign_out"
	"github.com/mewstcom/mewst/go/internal/handler/sign_up"
	"github.com/mewstcom/mewst/go/internal/middleware"
	"github.com/mewstcom/mewst/go/internal/ratelimit"
	"github.com/mewstcom/mewst/go/internal/repository"
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

	// リポジトリの初期化
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	actorRepo := repository.NewActorRepository(db)
	emailConfirmationRepo := repository.NewEmailConfirmationRepository(db)
	profileRepo := repository.NewProfileRepository(db)
	userProfileRepo := repository.NewUserProfileRepository(db)
	rateLimitRepo := repository.NewRateLimitRepository(db)

	// セッションマネージャーの初期化
	sessionMgr := session.NewManager(sessionRepo, actorRepo, userRepo, cfg)

	// メール送信クライアントの初期化
	var emailSender email.Sender
	if cfg.ResendAPIKey != "" && cfg.EmailFrom != "" {
		emailSender = email.NewResendSender(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
	} else {
		emailSender = email.NewNoopSender()
		slog.Warn("Resend APIキーまたは送信元メールアドレスが設定されていないため、メール送信は無効です")
	}

	// メール確認コード送信の初期化
	confirmationSender := email.NewConfirmationSender(emailSender)
	sendEmailConfirmationUC := usecase.NewSendEmailConfirmationUsecase(confirmationSender)

	// Workerの初期化
	workerClient, err := worker.NewClient(context.Background(), cfg.DatabaseDSN(), worker.Dependencies{
		SendEmailConfirmationUC: sendEmailConfirmationUC,
	})
	if err != nil {
		slog.Error("Workerクライアントの初期化に失敗しました", "error", err)
		os.Exit(1)
	}

	// Workerを開始
	if err := workerClient.Start(context.Background()); err != nil {
		slog.Error("Workerの開始に失敗しました", "error", err)
		os.Exit(1)
	}
	slog.Info("Workerを開始しました")

	// レートリミッターの初期化
	rateLimiter := ratelimit.NewLimiter(rateLimitRepo)

	// Dispatcherの初期化
	jobDispatcher := dispatcher.NewDispatcher(workerClient)

	// バリデーターの初期化
	signInValidator := validator.NewSignInCreateValidator(userRepo)
	signUpValidator := validator.NewSignUpCreateValidator(userRepo)
	emailConfirmationValidator := validator.NewEmailConfirmationCreateValidator(emailConfirmationRepo)
	accountsValidator := validator.NewAccountsCreateValidator(userRepo, profileRepo)
	passwordUpdateValidator := validator.NewPasswordUpdateValidator()
	passwordResetCreateValidator := validator.NewPasswordResetCreateValidator()

	// ユースケースの初期化
	createSessionUC := usecase.NewCreateSessionUsecase(actorRepo, sessionRepo)
	createSignInUC := usecase.NewCreateSignInUsecase(signInValidator, actorRepo, sessionRepo)
	createSignUpUC := usecase.NewCreateSignUpUsecase(signUpValidator, emailConfirmationRepo, jobDispatcher)
	createPasswordResetUC := usecase.NewCreatePasswordResetUsecase(passwordResetCreateValidator, emailConfirmationRepo, jobDispatcher)
	verifyEmailConfirmationUC := usecase.NewVerifyEmailConfirmationUsecase(emailConfirmationValidator, emailConfirmationRepo)
	getActiveEmailConfirmationUC := usecase.NewGetActiveEmailConfirmationUsecase(emailConfirmationRepo)
	getSucceededEmailConfirmationUC := usecase.NewGetSucceededEmailConfirmationUsecase(emailConfirmationRepo)
	updatePasswordUC := usecase.NewUpdatePasswordUsecase(passwordUpdateValidator, userRepo)
	createAccountUC := usecase.NewCreateAccountUsecase(db, accountsValidator, userRepo, profileRepo, userProfileRepo, actorRepo)

	// Turnstileクライアントの初期化
	turnstileClient := turnstile.NewClient(cfg.TurnstileSecretKey)

	// ハンドラーの初期化
	manifestHandler := manifest.NewHandler(cfg)
	signInHandler := sign_in.NewHandler(cfg, sessionMgr, createSignInUC, turnstileClient)
	signUpHandler := sign_up.NewHandler(cfg, sessionMgr, createSignUpUC, turnstileClient, rateLimiter)
	signOutHandler := sign_out.NewHandler(cfg, sessionMgr)
	passwordResetHandler := password_reset.NewHandler(cfg, sessionMgr, createPasswordResetUC, turnstileClient)
	emailConfirmationHandler := email_confirmation.NewHandler(cfg, sessionMgr, getActiveEmailConfirmationUC, verifyEmailConfirmationUC)
	passwordHandler := password.NewHandler(cfg, sessionMgr, getSucceededEmailConfirmationUC, updatePasswordUC)
	accountsHandler := accounts.NewHandler(cfg, sessionMgr, getSucceededEmailConfirmationUC, createAccountUC, createSessionUC, turnstileClient, rateLimiter)

	// ミドルウェアの初期化
	authMiddleware := middleware.NewAuth(sessionMgr)
	csrfMiddleware := middleware.NewCSRF(cfg)

	// Chiルーターの設定
	r := chi.NewRouter()

	// 基本ミドルウェア
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	// リバースプロキシの設定（Rails版へのプロキシ）
	if cfg.RailsAppURL != "" {
		proxyMiddleware, err := middleware.NewReverseProxyMiddleware(cfg.RailsAppURL, cfg)
		if err != nil {
			slog.Error("リバースプロキシミドルウェアの初期化に失敗しました", "error", err)
			os.Exit(1)
		}
		r.Use(proxyMiddleware.Middleware)
	}

	// 404ハンドラーの設定（ルーティングにマッチしないパス用）
	r.NotFound(handler.NotFound)

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
		r.Delete("/sign_out", signOutHandler.Delete)
		r.Post("/sign_out", signOutHandler.Delete)
	})

	// パスワードリセット・メール確認・アカウント作成（未認証ユーザーのみ）
	r.Group(func(r chi.Router) {
		r.Use(csrfMiddleware.Middleware)
		r.Use(authMiddleware.RequireNoAuth)

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
		r.Get("/accounts/new", accountsHandler.New)
		r.Post("/accounts", accountsHandler.Create)
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
