# typed: true
# frozen_string_literal: true

class Accounts::NewController < ApplicationController
  include Authenticatable
  include Localizable
  include PhoneNumberVerificationFindable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @account_activation = AccountActivation.new
    @account_activation.phone_number_verification = PhoneNumberVerification.find(session[:phone_number_verification_id])
  end
end
