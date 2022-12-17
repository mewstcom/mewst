# typed: true
# frozen_string_literal: true

class Users::NewController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_phone_number_verification_challenge_id
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    phone_number_verification_challenge = PhoneNumberVerificationChallenge.find(session[:phone_number_verification_challenge_id])
    @command = Commands::CreateUser.new(phone_number_verification_challenge:)
  end

  private

  sig { returns(T.untyped) }
  def require_phone_number_verification_challenge_id
    unless session[:phone_number_verification_challenge_id]
      redirect_to root_path
    end
  end
end
