# typed: true
# frozen_string_literal: true

class Latest::Timeline::ShowController < Latest::ApplicationController
  def call
    posts, page_info = current_profile!.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    render(json: Latest::Presenters::Timeline.new(posts:, page_info:))
  end
end
