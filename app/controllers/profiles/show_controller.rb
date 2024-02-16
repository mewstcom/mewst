# typed: true
# frozen_string_literal: true

class Profiles::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = Profile.kept.find_by!(atname: params[:atname])
    @follow_checker = FollowChecker.new(profile: current_actor&.profile, target_profiles: [@profile])

    result = Paginator.new(records: @profile.posts.kept).paginate(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 15
    )

    @posts = result.records
    @page_info = result.page_info
    @stamp_checker = StampChecker.new(profile: current_actor&.profile, posts: @posts)
  end
end
