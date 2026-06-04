# typed: strict
# frozen_string_literal: true

Sentry.init do |config|
  config.dsn = Rails.configuration.mewst["sentry_dsn"]
  config.breadcrumbs_logger = %i[active_support_logger http_logger]

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
