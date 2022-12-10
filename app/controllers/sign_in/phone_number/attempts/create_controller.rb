# typed: true
# frozen_string_literal: true

class SignIn::PhoneNumber::Attempts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])
    @verification_attempt = phone_number_verification.new_attempt(
      confirmation_code: verification_attempt_params[:confirmation_code]
    )

    if @verification_attempt.invalid?
      return render("sign_up/verification/attempts/new/call")
    end

    phone_number = PhoneNumber.find_by!(value: phone_number_verification.phone_number)

    ActiveRecord::Base.transaction do
      phone_number_verification.destroy
      sign_in(T.must(phone_number.user))
    end

    redirect_to home_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def verification_attempt_params
    T.cast(params.require(:phone_number_verification_attempt), ActionController::Parameters).permit(:confirmation_code)
  end
end
