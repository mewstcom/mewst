# typed: true
# frozen_string_literal: true

class Passwords::EditController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable
  include EmailConfirmationFindable

  around_action :set_locale
  before_action :require_no_authentication
  before_action :require_succeeded_email_confirmation

  sig { returns(T.untyped) }
  def call
    @form = EditPasswordForm.new
  end
end
