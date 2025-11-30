# typed: strict
# frozen_string_literal: true

class V1::StampNotificationItemResource < V1::ApplicationResource
  sig { params(notification: NotificationRecord, viewer: ActorRecord).void }
  def initialize(notification:, viewer:)
    @notification = notification
    @viewer = viewer
  end

  sig { returns(V1::ProfileResource) }
  def source_profile
    V1::ProfileResource.new(profile: notification.notifiable.profile_record, viewer:)
  end

  sig { returns(V1::PostResource) }
  def target_post
    V1::PostResource.new(post: notification.notifiable.post_record, viewer:)
  end

  sig { returns(NotificationRecord) }
  attr_reader :notification
  private :notification

  sig { returns(ActorRecord) }
  attr_reader :viewer
  private :viewer
end
