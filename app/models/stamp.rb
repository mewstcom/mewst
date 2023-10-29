# typed: strict
# frozen_string_literal: true

class Stamp < ApplicationRecord
  counter_culture :post

  belongs_to :post
  belongs_to :profile

  sig { void }
  def notify!
    stamp_notification = StampNotification.find_by(stamp: self)

    return if stamp_notification

    notification = Notification.create!(
      source_profile: profile,
      target_profile: post.profile,
      notifiable_type: NotifiableType::Stamp.serialize,
      notified_at: Time.current
    )
    StampNotification.create!(notification:, stamp: self)

    nil
  end

  sig { void }
  def unnotify!
    stamp_notification = StampNotification.find_by(stamp: self)

    return unless stamp_notification

    notification = stamp_notification.notification

    stamp_notification.delete
    notification.delete

    nil
  end
end
