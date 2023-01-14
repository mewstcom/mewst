# typed: true
# frozen_string_literal: true

class Settings::TwitterAuthorizations::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    authorization = TwitterAuthorization.new(form_params)
    authorization_url = authorization.url
    session[:code_verifier] = authorization.client.code_verifier

    redirect_to(authorization_url, allow_other_host: true)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:twitter_authorization), ActionController::Parameters).permit(:usage_cross_post, :usage_find_friends)
  end
end
