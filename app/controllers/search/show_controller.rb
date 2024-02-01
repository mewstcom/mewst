# typed: true
# frozen_string_literal: true

class Search::ShowController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    result = SuggestedProfileList.fetch(client: v1_public_client)

    @profiles = result.profiles
  end
end
