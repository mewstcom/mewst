# typed: true
# frozen_string_literal: true

class Passwords::EditController < ApplicationController
  include Authenticatable
  include Localizable
  include VerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_verification

  sig { returns(T.untyped) }
  def call
    @account = T.must(@verification).account
  end
end
