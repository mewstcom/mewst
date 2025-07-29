# typed: true
# frozen_string_literal: true

class Search::Profiles::IndexController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = KeywordSearchForm.new(q: params[:q].presence || "")
    page = ProfileRecord
      .kept
      .search_by_keywords(q: @form.q)
      .cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 15,
        order: {updated_at: :desc, id: :desc}
      )
      .fetch
    @profiles = page.records
    @page_info = PageInfo.from_cursor_paginate_page(page:)
    @follow_checker = FollowChecker.new(profile: viewer!.profile.not_nil!, target_profiles: @profiles)
  rescue ActiveRecordCursorPaginate::InvalidCursorError
    redirect_to(search_profile_list_path(q: params[:q]), status: :moved_permanently)
  end
end
