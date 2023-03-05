# typed: strict
# frozen_string_literal: true

module Pubsub::Subscribable
  extend T::Sig
  extend ActiveSupport::Concern

  sig { returns(T.untyped) }
  def require_authentication
    authenticate_or_request_with_http_token do |token|
      claim = Google::Auth::IDTokens.verify_oidc(token)
      claim["email_verified"] == true && claim["email"] == ENV.fetch("MEWST_GOOGLE_CLOUD_SERVICE_EMAIL")
    end
  end

  private

  def message_data
    @decoded_data ||= begin
      if (data = params.dig(:message, :data))
        JSON.parse(Base64.decode64(data))
      end
    end
  end
end

