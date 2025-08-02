# typed: strict
# frozen_string_literal: true

class SuggestedFollowRecord < ApplicationRecord
  self.table_name = "suggested_follows"

  belongs_to :source_profile_record, class_name: "ProfileRecord", foreign_key: :source_profile_id
  belongs_to :target_profile_record, class_name: "ProfileRecord", foreign_key: :target_profile_id

  scope :not_checked, -> { where(checked_at: nil) }
end
