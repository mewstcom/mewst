# typed: true
# frozen_string_literal: true

class Profiles::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale

  sig { returns(T.untyped) }
  def call
    @profile = ProfileRecord.kept.find_by!(atname: params[:atname])
    @follow_checker = FollowChecker.new(profile: viewer&.profile_record, target_profiles: [@profile])

    page = @profile.post_records.kept.preload(:link_record).cursor_paginate(
      after: params[:after].presence,
      before: params[:before].presence,
      limit: 15,
      order: {published_at: :desc, id: :desc}
    ).fetch

    @posts = page.records
    @page_info = PageInfo.from_cursor_paginate_page(page:)
    @stamp_checker = StampChecker.new(profile: viewer&.profile_record, posts: @posts)
  rescue ActiveRecordCursorPaginate::InvalidCursorError
    redirect_to(profile_path(@profile.atname), status: :moved_permanently)
  end
end
