# typed: true
# frozen_string_literal: true

class SignIn::Verification::PhoneNumber::Challenges::NewController < ApplicationController
  include Authenticatable
  include Localizable
  include PhoneNumberVerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_phone_number_verification_id

  sig { returns(T.untyped) }
  def call
    @challenge = PhoneNumberVerificationChallenge.new
  end
end
