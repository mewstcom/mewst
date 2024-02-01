# typed: true
# frozen_string_literal: true

class Notifications::IndexController < ApplicationController
  include Authenticatable
  include Localizable
  include ApiRequestable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    result = NotificationList.fetch(
      after: params[:after].presence,
      before: params[:before].presence,
      client: v1_public_client
    )

    @notifications = result.notifications
    @page_info = result.page_info
  end
end
