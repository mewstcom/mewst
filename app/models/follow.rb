# typed: strict
# frozen_string_literal: true

class Follow < ApplicationRecord
  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"

  sig { void }
  def notify!
    follow_notification = FollowNotification.find_by(source_profile:, target_profile:)

    return if follow_notification

    notification = Notification.create!(profile: target_profile, notifiable_type: :follow, notified_at: Time.current)
    FollowNotification.create!(notification:, source_profile:, target_profile:)

    nil
  end

  sig { void }
  def unnotify!
    follow_notification = FollowNotification.find_by(source_profile:, target_profile:)

    return unless follow_notification

    notification = follow_notification.notification

    follow_notification.delete
    notification.delete

    nil
  end
end
