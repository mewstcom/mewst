# typed: false
# frozen_string_literal: true

class Latest::NotificationSerializer < Latest::ApplicationSerializer
  root_key :notification, :notifications

  attributes :id, :kind, :notified_at

  one :source_profile, resource: Latest::ProfileSerializer

  one :item, resource: ->(resource) {
    case resource
    when Latest::FollowNotificationItemResource
      Latest::FollowNotificationItemSerializer
    when Latest::StampNotificationItemResource
      Latest::StampNotificationItemSerializer
    else
      raise "Unknown resource class: #{resource.class.inspect}"
    end
  }
end
