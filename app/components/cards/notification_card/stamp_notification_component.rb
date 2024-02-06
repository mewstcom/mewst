# typed: strict
# frozen_string_literal: true

class Cards::NotificationCard::StampNotificationComponent < ApplicationComponent
  sig { params(stamp_notification: StampNotification).void }
  def initialize(stamp_notification:)
    @stamp_notification = stamp_notification
  end

  sig { returns(StampNotification) }
  attr_reader :stamp_notification
  private :stamp_notification

  def target_post
    stamp_notification.post
  end
end
