# typed: strict
# frozen_string_literal: true

class Latest::FollowNotificationItemResource < Latest::ApplicationResource
  sig { params(follow_notification: FollowNotification, viewer: Actor).void }
  def initialize(follow_notification:, viewer:)
    @follow_notification = follow_notification
    @viewer = viewer
  end

  sig { returns(Latest::ProfileResource) }
  def profile
    Latest::ProfileResource.new(profile: follow_notification.source_profile, viewer:)
  end

  sig { returns(FollowNotification) }
  attr_reader :follow_notification
  private :follow_notification

  sig { returns(Actor) }
  attr_reader :viewer
  private :viewer
end
