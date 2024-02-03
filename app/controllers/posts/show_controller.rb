# typed: true
# frozen_string_literal: true

class Posts::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable
  include ResponseErrorable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    client = signed_in? ? v1_public_client : v1_internal_client

    @post = Post.fetch(client:, post_id: params[:post_id])
    not_found! unless @post

    @profile = @post.not_nil!.profile.not_nil!
    not_found! if @profile.atname != params[:atname]
  end
end
