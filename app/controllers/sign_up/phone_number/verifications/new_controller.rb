# typed: true
# frozen_string_literal: true

class SignUp::PhoneNumber::Verifications::NewController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_verification_challenge = PhoneNumberVerificationChallenge.find(session[:phone_number_verification_challenge_id])
    @command = Commands::VerifyPhoneNumber.new(phone_number_verification_challenge:)
  end
end
