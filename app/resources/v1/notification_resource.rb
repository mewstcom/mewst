# typed: strict
# frozen_string_literal: true

class V1::NotificationResource < V1::ApplicationResource
  delegate :id, to: :notification

  sig { params(notification: Notification, viewer: Actor).void }
  def initialize(notification:, viewer:)
    @notification = notification
    @viewer = viewer
  end

  sig { returns(V1::ProfileResource) }
  def source_profile
    V1::ProfileResource.new(profile: notification.source_profile.not_nil!, viewer:)
  end

  sig { returns(String) }
  def kind
    notification.notifiable_type.to_s
  end

  sig { returns(V1::StampNotificationItemResource) }
  def item
    case kind
    when NotifiableType::Stamp.serialize
      V1::StampNotificationItemResource.new(notification: notification, viewer:)
    else
      raise "Unknown notification kind: #{kind.inspect}"
    end
  end

  sig { returns(String) }
  def notified_at
    notification.notified_at.iso8601
  end

  sig { returns(Notification) }
  attr_reader :notification
  private :notification

  sig { returns(Actor) }
  attr_reader :viewer
  private :viewer
end
