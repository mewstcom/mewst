# typed: strict
# frozen_string_literal: true

class V1::StampNotificationItemResource < V1::ApplicationResource
  sig { params(stamp_notification: StampNotification, viewer: Actor).void }
  def initialize(stamp_notification:, viewer:)
    @stamp_notification = stamp_notification
    @viewer = viewer
  end

  sig { returns(V1::ProfileResource) }
  def source_profile
    V1::ProfileResource.new(profile: stamp_notification.profile, viewer:)
  end

  sig { returns(V1::PostResource) }
  def target_post
    V1::PostResource.new(post: stamp_notification.post, viewer:)
  end

  sig { returns(StampNotification) }
  attr_reader :stamp_notification
  private :stamp_notification

  sig { returns(Actor) }
  attr_reader :viewer
  private :viewer
end
