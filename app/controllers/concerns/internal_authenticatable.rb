# typed: strict
# frozen_string_literal: true

module InternalAuthenticatable
  extend T::Sig
  extend ActiveSupport::Concern

  include ActionController::HttpAuthentication::Token::ControllerMethods

  included do
    before_action :authenticate
  end

  sig { returns(T.nilable(Actor)) }
  def current_viewer
    nil
  end

  sig { void }
  private def authenticate
    authenticate_or_request_with_http_token do |token, _options|
      ActiveSupport::SecurityUtils.secure_compare(token, Rails.configuration.mewst["internal_api_token"])
    end
  end
end
