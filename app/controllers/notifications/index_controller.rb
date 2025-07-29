# typed: true
# frozen_string_literal: true

class Notifications::IndexController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    page = viewer!.profile_record.not_nil!
      .notification_records
      .preload(:source_profile_record, notifiable: {post_record: :profile_record})
      .cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 15,
        order: {notified_at: :desc, id: :desc}
      )
      .fetch
    @notifications = page.records
    @page_info = PageInfo.from_cursor_paginate_page(page:)
    @follow_checker = FollowChecker.new(
      profile: viewer!.profile_record.not_nil!,
      target_profiles: @notifications.map(&:source_profile_record)
    )
  rescue ActiveRecordCursorPaginate::InvalidCursorError
    redirect_to(notification_list_path, status: :moved_permanently)
  end
end
