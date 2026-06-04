# typed: strict
# frozen_string_literal: true

# Be sure to restart your server when you modify this file.

# Configure parameters to be partially matched (e.g. passw matches password) and filtered from the log file.
# Use this to limit dissemination of sensitive information.
# See the ActiveSupport::ParameterFilter documentation for supported notations and behaviors.
#
# Unlike the Rails default list, `:email` is included so that email addresses
# (PII) are masked in both Rails logs and the request parameters that
# sentry-rails attaches to breadcrumbs and events.
#
# [Ja] Rails のデフォルトリストと異なり `:email` を含めている。PII である
# メールアドレスを、Rails ログと sentry-rails が breadcrumbs / イベントに
# 添付するリクエストパラメータの両方でマスクするため。
Rails.application.config.filter_parameters += [
  :passw, :email, :secret, :token, :_key, :crypt, :salt, :certificate, :otp, :ssn
]
