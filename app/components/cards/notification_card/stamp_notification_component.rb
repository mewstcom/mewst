# typed: strict
# frozen_string_literal: true

class Cards::NotificationCard::StampNotificationComponent < ApplicationComponent
  sig { params(notification: NotificationRecord).void }
  def initialize(notification:)
    @notification = notification
  end

  sig { returns(NotificationRecord) }
  attr_reader :notification
  private :notification

  sig { returns(PostRecord) }
  def target_post
    notification.notifiable.post_record.not_nil!
  end
end
