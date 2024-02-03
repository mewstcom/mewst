# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @posts, @page_info = current_actor!.home_timeline.posts_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 15
    )
  end
end
