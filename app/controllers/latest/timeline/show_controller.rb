# typed: true
# frozen_string_literal: true

class Latest::Timeline::ShowController < Latest::ApplicationController
  def call
    profile = current_user!.profiles.find_by!(atname: params[:atname])
    posts, page_info = profile.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    json = Alba.serialize do
      attribute :posts do
        Resources::Latest::Post.new(posts).to_h
      end

      attribute :page_info do
        Resources::Latest::PageInfo.new(page_info).to_h
      end
    end

    render(json:)
  end
end
