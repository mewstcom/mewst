# typed: true
# frozen_string_literal: true

class SignIn::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @account = Account.find_by(email: form_params[:email])

    unless @account&.authenticate(form_params[:password])
      flash[:alert] = t("messages.accounts.authentication_failed")
      return redirect_to(sign_in_path)
    end

    sign_in(@account)

    flash[:success] = t("messages.accounts.signed_in_successfully")
    redirect_to home_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:account), ActionController::Parameters).permit(:email, :password)
  end
end
