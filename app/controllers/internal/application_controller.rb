# typed: strict
# frozen_string_literal: true

class Internal::ApplicationController < ActionController::API
  extend T::Sig

  include ActionController::HttpAuthentication::Token::ControllerMethods

  before_action :authenticate

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
