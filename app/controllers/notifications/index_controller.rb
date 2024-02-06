# typed: true
# frozen_string_literal: true

class Notifications::IndexController < ApplicationController
  include ControllerConcerns::Authenticatable
  include ControllerConcerns::Localizable

  around_action :set_locale
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    notifications = current_actor!.notifications.preload(:source_profile, stamp_notification: { stamp: { post: :profile } })
    result = Paginator.new(records: notifications).paginate(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 15,
      order_by: :notified_at
    )

    @notifications = result.records
    @page_info = result.page_info
  end
end
