# typed: true
# frozen_string_literal: true

class SignUp::PhoneNumber::Attempts::NewController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])
    @verification_attempt = phone_number_verification.new_attempt
  end
end
