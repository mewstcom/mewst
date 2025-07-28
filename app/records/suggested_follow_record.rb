# typed: strict
# frozen_string_literal: true

class SuggestedFollowRecord < ApplicationRecord
  self.table_name = "suggested_follows"

  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"

  scope :not_checked, -> { where(checked_at: nil) }
end