# typed: true
# frozen_string_literal: true

class Profiles::ShowController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable
  include ResponseErrorable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    client = signed_in? ? v1_public_client : v1_internal_client
    profile_post = ProfilePost.fetch(
      client:,
      atname: params[:atname],
      before: params[:before].presence,
      after: params[:after].presence
    )

    @profile = profile_post.profile
    not_found! unless @profile

    @posts = profile_post.posts
    @page_info = profile_post.page_info
  end
end
