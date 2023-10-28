# typed: strict
# frozen_string_literal: true

class Stamp < ApplicationRecord
  counter_culture :post

  belongs_to :post
  belongs_to :profile

  sig { void }
  def notify!
    follow_notification = FollowNotification.find_by(source_profile:, target_profile:)

    return if follow_notification

    notification = Notification.create!(profile: target_profile, notifiable_type: :follow, notified_at: Time.current)
    FollowNotification.create!(notification:, source_profile:, target_profile:)

    nil
  end
end
