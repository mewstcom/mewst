# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @form = PostForm.new(with_frame: true)
    @posts, @page_info = viewer!.profile_record.not_nil!.home_timeline.fetch_posts(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 15
    )
    @stamp_checker = StampChecker.new(profile: viewer!.profile_record, posts: @posts)
  end
end
