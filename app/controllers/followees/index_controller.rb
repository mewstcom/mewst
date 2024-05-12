# typed: true
# frozen_string_literal: true

class Followees::IndexController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    page = viewer!.followees.kept.cursor_paginate(
      after: params[:after].presence,
      before: params[:before].presence,
      limit: 15,
      order: {Arel.sql("follows.followed_at") => :desc, id: :desc}
    ).fetch
    @followees = page.records
    @page_info = PageInfo.from_cursor_paginate_page(page:)
  end
end
