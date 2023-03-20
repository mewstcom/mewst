# typed: true
# frozen_string_literal: true

class Accounts::NewController < ApplicationController
  include Authenticatable
  include Localizable
  include VerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_verification_id

  sig { returns(T.untyped) }
  def call
    @account_activation = AccountActivation.new
    @account_activation.verification = Verification.find(session[:verification_id])
  end
end
