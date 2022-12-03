# typed: true
# frozen_string_literal: true

class PhoneNumbers::NewController < ApplicationController
  include Authenticatable

  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = PhoneNumberForm.new
  end
end
