# typed: true
# frozen_string_literal: true

class SignIn::PhoneNumber::Attempts::NewController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])
    @verification_attempt = phone_number_verification.new_attempt
  end
end
