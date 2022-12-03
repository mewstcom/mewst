# typed: true
# frozen_string_literal: true

class SignUp::Confirmations::NewController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = VerificationCodeForm.new
  end
end
