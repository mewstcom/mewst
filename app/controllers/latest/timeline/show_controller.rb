# typed: true
# frozen_string_literal: true

class Latest::Timeline::ShowController < Latest::ApplicationController
  def call
    posts, page_info = current_profile!.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    post_entities = posts.map { |post| Latest::Entities::Post.new(post:) }
    render(json: {
      posts: Latest::Resources::Post.new(post_entities).to_h,
      page_info: Latest::Resources::PageInfo.new(page_info).to_h
    })
  end
end
