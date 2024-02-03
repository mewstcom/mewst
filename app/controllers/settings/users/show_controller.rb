# typed: true
# frozen_string_literal: true

class Settings::Users::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    user = User.fetch_me(client: v1_public_client)

    @form = UserForm.new(
      locale: user.locale.not_nil!.serialize
    )
  end
end
