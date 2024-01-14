# typed: false
# frozen_string_literal: true

class V1::NotificationSerializer < V1::ApplicationSerializer
  root_key :notification, :notifications

  attributes :id, :kind, :notified_at

  one :source_profile, resource: V1::ProfileSerializer

  one :item, resource: ->(resource) {
    case resource
    when V1::StampNotificationItemResource
      V1::StampNotificationItemSerializer
    else
      raise "Unknown resource class: #{resource.class.inspect}"
    end
  }
end
