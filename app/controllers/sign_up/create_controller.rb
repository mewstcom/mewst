# typed: true
# frozen_string_literal: true

class SignUp::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number_verification = PhoneNumberVerification.new(phone_number_verification_params)
    @phone_number_verification.set_confirmation_code

    if @phone_number_verification.invalid?
      return render("sign_up/new/call")
    end

    @phone_number_verification.send_sms

    session[:phone_number_verification_id] = @phone_number_verification.id
    flash[:success] = t("messages.authentication.confirmation_sms_sent")
    redirect_to sign_up_phone_number_new_attempt_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def phone_number_verification_params
    T.cast(params.require(:phone_number_verification), ActionController::Parameters).permit(:phone_number, :phone_number_origin)
  end
end
