# typed: true
# frozen_string_literal: true

class Settings::TwitterAuthorizations::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    twitter_auth = TwitterAuthorization.new(form_params)

    authorization_uri = twitter_auth.authorization_uri
    session[:twitter_oauth2_code_verifier] = twitter_auth.code_verifier
    session[:twitter_oauth2_state] = twitter_auth.state

    redirect_to(authorization_uri, allow_other_host: true)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:twitter_authorization), ActionController::Parameters)
      .permit(:cross_post_usage, :find_friends_usage)
  end
end
