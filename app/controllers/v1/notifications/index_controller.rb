# typed: true
# frozen_string_literal: true

class V1::Notifications::IndexController < V1::ApplicationController
  include PublicAuthenticatable

  def call
    notifications = current_viewer!.notifications.preload(:stamp_notification)
    result = Paginator.new(records: notifications).paginate(
      before: params[:before].presence,
      after: params[:after].presence,
      limit: 15,
      order_by: :notified_at
    )

    notification_resources = result.records.map { |notification| V1::NotificationResource.new(notification:, viewer: current_viewer!) }
    render(
      json: {
        notifications: V1::NotificationSerializer.new(notification_resources).to_h,
        page_info: V1::PageInfoSerializer.new(result.page_info).to_h
      }
    )
  end
end
