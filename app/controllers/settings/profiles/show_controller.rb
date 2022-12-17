# typed: true
# frozen_string_literal: true

class Settings::Profiles::ShowController < ApplicationController
  include Authenticatable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @profile = T.must(current_profile)
  end
end
