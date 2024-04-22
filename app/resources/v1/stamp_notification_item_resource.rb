# typed: strict
# frozen_string_literal: true

class V1::StampNotificationItemResource < V1::ApplicationResource
  sig { params(notification: Notification, viewer: Actor).void }
  def initialize(notification:, viewer:)
    @notification = notification
    @viewer = viewer
  end

  sig { returns(V1::ProfileResource) }
  def source_profile
    V1::ProfileResource.new(profile: notification.notifiable.profile, viewer:)
  end

  sig { returns(V1::PostResource) }
  def target_post
    V1::PostResource.new(post: notification.notifiable.post, viewer:)
  end

  sig { returns(Notification) }
  attr_reader :notification
  private :notification

  sig { returns(Actor) }
  attr_reader :viewer
  private :viewer
end
