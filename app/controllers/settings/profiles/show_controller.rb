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
      name: current_actor!.name,
      description: current_actor!.description,
      avatar_kind: current_actor!.avatar_kind,
      gravatar_email: current_actor!.gravatar_email,
      image_url: current_actor!.image_url
    )
  end
end
