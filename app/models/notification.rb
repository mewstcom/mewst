# typed: strict
# frozen_string_literal: true

class Notification < ApplicationRecord
  extend Enumerize

  enumerize :notifiable_type, in: NotifiableType.values.map(&:serialize)

  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"
  has_one :stamp_notification, dependent: :restrict_with_exception

  def deserialized_notifiable_type
    @deserialized_notifiable_type ||= NotifiableType.deserialize(notifiable_type)
  end
end
