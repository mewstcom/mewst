# typed: true
# frozen_string_literal: true

class Latest::Timeline::ShowController < Latest::ApplicationController
  def call
    posts, page_info = current_profile.not_nil!.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    resources = posts.map { |post| Latest::PostResource.new(post:, viewer: current_profile.not_nil!) }
    render(
      json: {
        posts: Latest::PostSerializer.new(resources).to_h,
        page_info: Latest::PageInfoSerializer.new(page_info).to_h
      }
    )
  end
end
