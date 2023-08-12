# typed: true
# frozen_string_literal: true

class Latest::Timeline::ShowController < Latest::ApplicationController
  def call
    posts, page_info = current_profile!.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    resources = posts.map { |post| Latest::PostResource.new(post:, viewer: current_profile!) }
    render(
      json: Panko::Response.new(
        posts: Panko::ArraySerializer.new(resources, each_serializer: Latest::PostSerializer),
        page_info: Latest::PageInfoSerializer.new.serialize(page_info)
      )
    )
  end
end
