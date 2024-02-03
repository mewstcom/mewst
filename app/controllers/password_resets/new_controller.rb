# typed: true
# frozen_string_literal: true

class PasswordResets::NewController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_no_authentication

  sig { returns(T.untyped) }
  def call
    @form = EmailConfirmationForm.new
  end
end
