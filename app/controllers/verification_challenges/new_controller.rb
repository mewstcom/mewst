# typed: true
# frozen_string_literal: true

class VerificationChallenges::NewController < ApplicationController
  include Authenticatable
  include Localizable
  include VerificationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_verification_id

  sig { returns(T.untyped) }
  def call
    @challenge = VerificationChallenge.new
  end
end
