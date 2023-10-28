# typed: strict
# frozen_string_literal: true

class Notification < ApplicationRecord
  extend Enumerize

  counter_culture :profile, column_name: :unread_notifications_count

  enumerize :notifiable_type, in: %i[follow stamp]

  belongs_to :profile
  has_one :follow_notification, dependent: :restrict_with_exception
  has_one :stamp_notification, dependent: :restrict_with_exception
end
