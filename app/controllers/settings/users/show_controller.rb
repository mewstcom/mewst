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
      locale: current_actor!.locale.serialize,
      time_zone: current_actor!.time_zone
    )
  end
end
