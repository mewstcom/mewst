# typed: strict
# frozen_string_literal: true

class Notification < ApplicationRecord
  extend Enumerize

  delegated_type :notifiable, types: NotifiableType.values.map(&:serialize)

  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"

  sig { returns(NotifiableType) }
  def deserialized_notifiable_type
    NotifiableType.deserialize(notifiable_type)
  end
end
