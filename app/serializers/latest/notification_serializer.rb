# typed: false
# frozen_string_literal: true

class Latest::NotificationSerializer < Latest::ApplicationSerializer
  root_key :notification, :notifications

  attributes :id, :kind, :notified_at

  one :item, resource: ->(resource) {
    case resource
    when Latest::FollowNotificationItemResource
      Latest::FollowNotificationItemSerializer
    else
      raise "Unknown resource class: #{resource.class.inspect}"
    end
  }
end
