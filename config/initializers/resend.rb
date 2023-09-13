# typed: strict
# frozen_string_literal: true

if Rails.env.production?
  Resend.api_key = Rails.configuration.mewst["resend_api_key"]
end
