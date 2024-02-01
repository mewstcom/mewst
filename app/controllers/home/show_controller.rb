# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    result = HomeTimeline.fetch(
      after: params[:after].presence,
      before: params[:before].presence,
      client: v1_public_client
    )

    @posts = result.posts
    @page_info = result.page_info
  end
end
