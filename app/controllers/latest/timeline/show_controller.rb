# typed: true
# frozen_string_literal: true

class Latest::Timeline::ShowController < Latest::ApplicationController
  include Paginateable
  def call
    profile = current_user!.profiles.find_by!(atname: params[:atname])
    posts, page_info = profile.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    response_pagination_headers(page_info:, path: "/@#{params[:atname]}/timeline")
    render(json: Resources::Latest::Post.new(posts))
  end
end
