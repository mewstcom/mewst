# typed: true
# frozen_string_literal: true

class Home::ShowController < ApplicationController
  include Authenticatable
  include Authorizable
  include Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    @entries, @page_info = current_profile!.home_timeline.entries_with_page_info(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 20
    )
  end
end
