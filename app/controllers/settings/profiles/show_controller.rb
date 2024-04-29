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
      viewer: viewer,
      atname: viewer!.atname,
      name: viewer!.name,
      description: viewer!.description,
      avatar_kind: viewer!.avatar_kind,
      gravatar_email: viewer!.gravatar_email,
      image_url: viewer!.image_url
    )
  end
end
