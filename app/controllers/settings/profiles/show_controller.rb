# typed: true
# frozen_string_literal: true

class Settings::Profiles::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    profile = Profile.fetch_me(client: v1_public_client)

    @form = ProfileForm.new(
      atname: profile.atname,
      avatar_url: profile.avatar_url,
      description: profile.description,
      name: profile.name
    )
  end
end
