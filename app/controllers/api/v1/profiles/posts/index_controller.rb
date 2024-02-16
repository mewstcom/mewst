# typed: true
# frozen_string_literal: true

class Api::V1::Profiles::Posts::IndexController < Api::V1::ApplicationController
  # include ControllerConcerns::PublicAuthenticatable
  #
  # def call
  #   profile = Profile.kept.find_by!(atname: params[:atname])
  #   result = Paginator.new(records: profile.posts.kept).paginate(
  #     before: params[:before].presence,
  #     after: params[:after].presence,
  #     limit: 15
  #   )
  #
  #   profile_resource = V1::ProfileResource.new(profile:, viewer: current_viewer)
  #   post_resources = result.records.map { |post| V1::PostResource.new(post:, viewer: current_viewer) }
  #
  #   render(
  #     json: {
  #       profile: V1::ProfileSerializer.new(profile_resource).to_h,
  #       posts: V1::PostSerializer.new(post_resources).to_h,
  #       page_info: V1::PageInfoSerializer.new(result.page_info).to_h
  #     }
  #   )
  # end
end
