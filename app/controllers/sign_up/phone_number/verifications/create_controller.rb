# typed: true
# frozen_string_literal: true

class SignUp::PhoneNumber::Verifications::CreateController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_verification_challenge = PhoneNumberVerificationChallenge.find(session[:phone_number_verification_challenge_id])
    confirmation_code, = form_params.values_at(:confirmation_code)
    @command = Commands::VerifyPhoneNumber.new(phone_number_verification_challenge:, confirmation_code:)

    if @command.invalid?
      return render("sign_up/phone_number/verifications/new/call")
    end

    redirect_to new_user_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:commands_verify_phone_number), ActionController::Parameters).permit(:confirmation_code)
  end
end
