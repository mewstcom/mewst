# typed: true
# frozen_string_literal: true

class Api::V1::Timeline::ShowController < Api::V1::ApplicationController
  # include ControllerConcerns::PublicAuthenticatable
  #
  # def call
  #   posts, page_info = current_viewer!.home_timeline.posts_with_page_info(
  #     before: params[:before].presence,
  #     after: params[:after].presence,
  #     limit: 15
  #   )
  #
  #   resources = posts.map { |post| V1::PostResource.new(post:, viewer: current_viewer!) }
  #   render(
  #     json: {
  #       posts: V1::PostSerializer.new(resources).to_h,
  #       page_info: V1::PageInfoSerializer.new(page_info).to_h
  #     }
  #   )
  # end
end
