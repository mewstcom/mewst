# typed: true
# frozen_string_literal: true

class SignIn::Verification::PhoneNumber::Challenges::CreateController < ApplicationController
  include Authenticatable
  include Localizable
  include PhoneNumberVerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_phone_number_verification_id

  sig { returns(T.untyped) }
  def call
    @challenge = PhoneNumberVerificationChallenge.new(form_params)
    @challenge.phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])

    if @challenge.invalid?
      return render("sign_in/verification/phone_number/challenges/new/call", status: :unprocessable_entity)
    end

    account = @challenge.account

    unless account
      return redirect_to(new_account_path)
    end

    ActiveRecord::Base.transaction do
      sign_in(account)
      @challenge.phone_number_verification!.destroy
    end

    redirect_to home_path
  end

  private

  sig { returns(ActionController::Parameters) }
  def form_params
    T.cast(params.require(:phone_number_verification_challenge), ActionController::Parameters).permit(:challenged_confirmation_code)
  end
end
