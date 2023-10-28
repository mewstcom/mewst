# typed: true
# frozen_string_literal: true

class Latest::Notifications::IndexController < Latest::ApplicationController
  include Latest::Authenticatable

  def call
    notifications = current_viewer!.notifications.preload(:follow_notification, :stamp_notification)
    result = Paginator.new(records: notifications).paginate(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 5,
      order_by: :notified_at
    )

    notification_resources = result.records.map { |notification| Latest::NotificationResource.new(notification:, viewer: current_viewer!) }
    render(
      json: {
        notifications: Latest::NotificationSerializer.new(notification_resources).to_h,
        page_info: Latest::PageInfoSerializer.new(result.page_info).to_h
      }
    )
  end
end
