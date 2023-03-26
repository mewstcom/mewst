# typed: true
# frozen_string_literal: true

class Passwords::UpdateController < ApplicationController
  include Authenticatable
  include Localizable
  include VerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_verification

  sig { returns(T.untyped) }
  def call
    @account = @verification.account

    if @account.update(form_params)
      reset_session
      flash[:success] = t("messages.accounts.password_reset_successfully")
      redirect_to sign_in_path
    else
      render("passwords/edit/call", status: :unprocessable_entity)
    end
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:account), ActionController::Parameters).permit(:password)
  end
end
