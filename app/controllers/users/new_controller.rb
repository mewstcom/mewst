# typed: true
# frozen_string_literal: true

class Users::NewController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_phone_number_verification_id
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])
    @user_creator = @phone_number_verification.new_user_creator
  end

  private

  sig { returns(T.untyped) }
  def require_phone_number_verification_id
    unless session[:phone_number_verification_id]
      redirect_to root_path
    end
  end
end
