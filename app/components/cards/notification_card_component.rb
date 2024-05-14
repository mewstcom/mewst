# typed: strict
# frozen_string_literal: true

class Cards::NotificationCardComponent < ApplicationComponent
  sig { params(notification: Notification).void }
  def initialize(notification:)
    @notification = notification
  end

  sig { returns(Notification) }
  attr_reader :notification
  private :notification

  sig { returns(Profile) }
  def source_profile
    notification.source_profile.not_nil!
  end
end
