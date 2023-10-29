# typed: strict
# frozen_string_literal: true

class Latest::StampNotificationItemResource < Latest::ApplicationResource
  sig { params(stamp_notification: StampNotification, viewer: Actor).void }
  def initialize(stamp_notification:, viewer:)
    @stamp_notification = stamp_notification
    @viewer = viewer
  end

  sig { returns(Latest::ProfileResource) }
  def source_profile
    Latest::ProfileResource.new(profile: stamp_notification.profile, viewer:)
  end

  sig { returns(Latest::PostResource) }
  def target_post
    Latest::PostResource.new(post: stamp_notification.post, viewer:)
  end

  sig { returns(StampNotification) }
  attr_reader :stamp_notification
  private :stamp_notification

  sig { returns(Actor) }
  attr_reader :viewer
  private :viewer
end
