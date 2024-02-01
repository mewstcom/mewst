# typed: strict
# frozen_string_literal: true

class Cards::NotificationCard::StampNotificationComponent < ApplicationComponent
  sig { params(notification_item: StampNotificationItem).void }
  def initialize(notification_item:)
    @notification_item = notification_item
  end

  sig { returns(StampNotificationItem) }
  attr_reader :notification_item
  private :notification_item
end
