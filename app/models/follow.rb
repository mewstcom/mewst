# typed: strict
# frozen_string_literal: true

class Follow < ApplicationRecord
  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"

  sig { void }
  def notify!
    follow_notification = FollowNotification.find_by(follow: self)

    return if follow_notification

    notification = Notification.create!(
      source_profile:,
      target_profile:,
      notifiable_type: NotifiableType::Follow.serialize,
      notified_at: Time.current
    )
    FollowNotification.create!(notification:, follow: self)

    nil
  end

  sig { void }
  def unnotify!
    follow_notification = FollowNotification.find_by(follow: self)

    return unless follow_notification

    notification = follow_notification.notification

    follow_notification.delete
    notification.delete

    nil
  end
end
