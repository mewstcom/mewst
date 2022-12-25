# typed: true
# frozen_string_literal: true

class Accounts::CreateController < ApplicationController
  include Authenticatable
  include Localizable
  include PhoneNumberVerificationFindable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    locale = I18n.locale.to_s
    @account_activation = Account::Activation.new(form_params.merge(locale:))
    @account_activation.phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])

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
    T.cast(params.require(:account_activation), ActionController::Parameters).permit(:atname)
  end
end
