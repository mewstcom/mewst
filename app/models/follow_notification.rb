# typed: strict
# frozen_string_literal: true

class FollowNotification < ApplicationRecord
  belongs_to :notification
  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"
end
