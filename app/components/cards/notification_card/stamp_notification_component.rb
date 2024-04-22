# typed: strict
# frozen_string_literal: true

class Cards::NotificationCard::StampNotificationComponent < ApplicationComponent
  sig { params(notification: Notification).void }
  def initialize(notification:)
    @notification = notification
  end

  sig { returns(Notification) }
  attr_reader :notification
  private :notification

  sig { returns(Post) }
  def target_post
    notification.notifiable.post.not_nil!
  end
end
