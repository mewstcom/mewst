# typed: true
# frozen_string_literal: true

class Accounts::CreateController < ApplicationController
  include Authenticatable
  include Localizable
  include VerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_verification

  sig { returns(T.untyped) }
  def call
    @account_activation = AccountActivation.new(form_params.merge(locale: I18n.locale.to_s))
    @account_activation.verification = @verification

    account = @account_activation.run

    reset_session
    sign_in(account)

    redirect_to home_path
  rescue ActiveModel::ValidationError
    render("accounts/new/call", status: :unprocessable_entity)
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:account_activation), ActionController::Parameters).permit(:atname, :password)
  end
end
