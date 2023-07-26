# typed: strict
# frozen_string_literal: true

module Pubsub::Subscribable
  extend T::Sig
  extend ActiveSupport::Concern

  included do
    include ActionController::HttpAuthentication::Token::ControllerMethods
  end

  sig { void }
  def require_authentication
    authenticate_or_request_with_http_token do |token|
      claim = Google::Auth::IDTokens.verify_oidc(token)
      claim["email_verified"] == true && claim["email"] == Rails.configuration.mewst["google_cloud_service_email"]
    end
  end

  sig { returns(T.nilable(T::Hash[String, String])) }
  def message_data
    if (data = params.dig(:message, :data))
      JSON.parse(Base64.decode64(data))
    end
  end
end
