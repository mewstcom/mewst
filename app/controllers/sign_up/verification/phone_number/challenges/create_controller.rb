# typed: true
# frozen_string_literal: true

class SignUp::Verification::PhoneNumber::Challenges::CreateController < ApplicationController
  include Authenticatable
  include Localizable
  include PhoneNumberVerificationFindable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @challenge = PhoneNumberVerificationChallenge.new(form_params)
    @challenge.phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])

    if @challenge.invalid?
      return render("sign_up/verification/phone_number/challenges/new/call", status: :unprocessable_entity)
    end

    redirect_to new_account_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:phone_number_verification_challenge), ActionController::Parameters).permit(:challenged_confirmation_code)
  end
end
