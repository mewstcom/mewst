# typed: strict
# frozen_string_literal: true

class Cards::NotificationCardComponent < ApplicationComponent
  sig { params(notification: NotificationRecord, follow_checker: FollowChecker).void }
  def initialize(notification:, follow_checker:)
    @notification = notification
    @follow_checker = follow_checker
  end

  sig { returns(NotificationRecord) }
  attr_reader :notification
  private :notification

  sig { returns(FollowChecker) }
  attr_reader :follow_checker
  private :follow_checker

  sig { returns(ProfileRecord) }
  def source_profile
    notification.source_profile_record.not_nil!
  end
end
