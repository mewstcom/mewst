# typed: true
# frozen_string_literal: true

class Internal::Profiles::Posts::IndexController < Internal::ApplicationController
  def call
    profile = Profile.find_by!(atname: params[:atname])
    result = Paginator.new(records: profile.posts).paginate(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5
    )

    profile_resource = Latest::ProfileResource.new(profile:, viewer: nil)
    post_resources = result.records.map { |post| Latest::PostResource.new(post:, viewer: nil) }

    render(
      json: {
        profile: Latest::ProfileSerializer.new(profile_resource).to_h,
        posts: Latest::PostSerializer.new(post_resources).to_h,
        page_info: Latest::PageInfoSerializer.new(result.page_info).to_h
      }
    )
  end
end
