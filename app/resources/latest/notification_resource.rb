# typed: strict
# frozen_string_literal: true

class Latest::NotificationResource < Latest::ApplicationResource
  delegate :id, to: :notification

  sig { params(notification: Notification, viewer: Actor).void }
  def initialize(notification:, viewer:)
    @notification = notification
    @viewer = viewer
  end

  sig { returns(String) }
  def kind
    notification.notifiable_type.to_s
  end

  sig { returns(T.any(Latest::FollowNotificationItemResource, Latest::StampNotificationItemResource)) }
  def item
    case kind
    when "follow"
      Latest::FollowNotificationItemResource.new(follow_notification: notification.follow_notification, viewer:)
    when "stamp"
      Latest::StampNotificationItemResource.new(stamp_notification: notification.stamp_notification, viewer:)
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
