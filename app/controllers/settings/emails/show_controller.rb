# typed: true
# frozen_string_literal: true

class Settings::Emails::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = EmailUpdateForm.new
  end
end
