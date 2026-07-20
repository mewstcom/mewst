# typed: strict
# frozen_string_literal: true

Sentry.init do |config|
  config.dsn = Rails.configuration.mewst["sentry_dsn"]
  config.breadcrumbs_logger = %i[active_support_logger http_logger]

  # Tag events with the deployed commit hash (GIT_REV, provided by Dokku) so error
  # grouping respects release boundaries and Sentry's "resolve in the next release"
  # workflow can tell deploys apart. Skip when missing to keep Sentry's
  # auto-detection from being overridden with an empty string.
  #
  # [Ja] デプロイ単位でエラーを分離し「次のリリースで resolve」を機能させるため、
  # リリースタグに Dokku 提供のコミットハッシュ (GIT_REV) を設定する。値が空の場合は
  # Sentry の自動検出を空文字で上書きしないよう設定自体を行わない。
  git_rev = Rails.configuration.mewst["git_rev"].presence
  config.release = git_rev if git_rev

  # Set traces_sample_rate to 1.0 to capture 100%
  # of transactions for performance monitoring.
  # We recommend adjusting this value in production.
  config.traces_sample_rate = 0.5

  # Set profiles_sample_rate to profile 100%
  # of sampled transactions.
  # We recommend adjusting this value in production.
  config.profiles_sample_rate = 0.5

  # Pair with the SentryEventFilter below: never auto-attach PII so the scrub
  # only has to defend against accidentally-captured request payloads.
  #
  # [Ja] 後段の SentryEventFilter と合わせて、PII を自動添付しない方針を明示する。
  # 想定外のリクエストペイロード捕捉に対する多層防御として動作する。
  config.send_default_pii = false

  config.before_send = ->(event, hint) { SentryEventFilter.call(event, hint) }
end
