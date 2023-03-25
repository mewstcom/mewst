# typed: true
# frozen_string_literal: true

class Settings::Accounts::UpdateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @account = current_account!
    @account.attributes = form_params

    if @account.save
      flash[:success] = t("messages.accounts.updated")
      redirect_to settings_account_path
    else
      render("settings/accounts/show/call", status: :unprocessable_entity)
    end
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:account), ActionController::Parameters).permit(:locale)
  end
end
