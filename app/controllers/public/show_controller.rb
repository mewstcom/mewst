# typed: true
# frozen_string_literal: true

class Public::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = PostForm.new(with_frame: true)
    page = Post.kept.preload(:profile, :link).cursor_paginate(
      after: params[:after].presence,
      before: params[:before].presence,
      limit: 15,
      order: {published_at: :desc, id: :desc}
    ).fetch

    @posts = page.records
    @page_info = PageInfo.from_cursor_paginate_page(page:)
    @stamp_checker = StampChecker.new(profile: viewer!.profile, posts: @posts)
  rescue ActiveRecordCursorPaginate::InvalidCursorError
    redirect_to(public_path, status: :moved_permanently)
  end
end
