# typed: strict
# frozen_string_literal: true

class Notification < ApplicationRecord
  extend Enumerize

  counter_culture :target_profile, column_name: :unread_notifications_count

  enumerize :notifiable_type, in: NotifiableType.values.map(&:serialize)

  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"
  has_one :follow_notification, dependent: :restrict_with_exception
  has_one :stamp_notification, dependent: :restrict_with_exception
end
