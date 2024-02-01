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
end
