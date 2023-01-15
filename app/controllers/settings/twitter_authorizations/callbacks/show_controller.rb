# typed: true
# frozen_string_literal: true

class Settings::TwitterAuthorizations::Callbacks::ShowController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    if invalid_response?
      flash[:alert] = t("messages.connected_accounts.connecting_error")
      return redirect_to(settings_connected_account_list_path)
    end

    current_profile!.upsert_twitter_account(authorization_code:, code_verifier:)

    flash[:success] = t("messages.connected_accounts.connected")
    redirect_to settings_connected_account_list_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def callback_params
    T.cast(params, ActionController::Parameters).permit(:code, :error, :state)
  end

  def invalid_response?
    callback_params[:error] ||
      callback_params[:state] != session[:twitter_oauth2_state] ||
      !code_verifier
  end

  def authorization_code
    callback_params[:code]
  end

  def code_verifier
    session[:twitter_oauth2_code_verifier]
  end
end
