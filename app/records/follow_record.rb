# typed: strict
# frozen_string_literal: true

class FollowRecord < ApplicationRecord
  self.table_name = "follows"

  belongs_to :source_profile_record, class_name: "ProfileRecord", foreign_key: :source_profile_id
  belongs_to :target_profile_record, class_name: "ProfileRecord", foreign_key: :target_profile_id

  sig { void }
  def check_suggested!
    SuggestedFollowRecord.find_by(source_profile_id: source_profile_id, target_profile_id: target_profile_id)&.touch(:checked_at)
  end
end
