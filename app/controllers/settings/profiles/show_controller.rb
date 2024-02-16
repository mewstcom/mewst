# typed: true
# frozen_string_literal: true

class Settings::Profiles::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = ProfileForm.new(
      viewer: current_actor,
      atname: current_actor!.atname,
      avatar_url: current_actor!.avatar_url,
      description: current_actor!.description,
      name: current_actor!.name
    )
  end
end
