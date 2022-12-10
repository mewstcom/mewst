# typed: true
# frozen_string_literal: true

class SignIn::NewController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number_verification = PhoneNumberVerification.new
  end
end
