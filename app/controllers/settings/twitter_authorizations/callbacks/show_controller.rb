# typed: true
# frozen_string_literal: true

class Settings::TwitterAuthorizations::Callbacks::ShowController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    TwitterAccount.create_from_authorization_code(code: form_params[:code])

    flash[:success] = t("messages.twitter_accounts.connected")
    redirect_back(fallback_location: settings_connected_account_list_path)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params, ActionController::Parameters).permit(:code)
  end
end
