# typed: true
# frozen_string_literal: true

class Latest::Timeline::ShowController < Latest::ApplicationController
  def call
    profile = current_user.profiles.find_by!(atname: params[:atname])
    @posts, @page_info = profile.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 20
    )
    @posts = Post.preload(:postable, :profile).first(3)

    render(json: { data: Resources::Post.new(@posts) }.as_json)
  end
end
