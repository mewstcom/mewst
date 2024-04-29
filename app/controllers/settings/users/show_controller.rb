# typed: true
# frozen_string_literal: true

class Settings::Users::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = UserForm.new(
      locale: viewer!.locale.serialize,
      time_zone: viewer!.time_zone
    )
  end
end
