# typed: true
# frozen_string_literal: true

class Latest::Profiles::Posts::IndexController < Latest::ApplicationController
  include Latest::Authenticatable

  def call
    profile = Profile.kept.find_by!(atname: params[:atname])
    result = Paginator.new(records: profile.posts.kept).paginate(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 15
    )

    profile_resource = Latest::ProfileResource.new(profile:, viewer: current_viewer)
    post_resources = result.records.map { |post| Latest::PostResource.new(post:, viewer: current_viewer) }

    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h,
        posts: Latest::PostSerializer.new(post_resources).to_h,
        page_info: Latest::PageInfoSerializer.new(result.page_info).to_h
      }
    )
  end
end
