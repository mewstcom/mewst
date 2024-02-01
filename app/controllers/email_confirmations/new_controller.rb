# typed: true
# frozen_string_literal: true

class EmailConfirmations::NewController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable
  include EmailConfirmationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_email_confirmation_id

  sig { returns(T.untyped) }
  def call
    @form = EmailConfirmationChallengeForm.new
  end
end
