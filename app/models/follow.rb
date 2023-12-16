# typed: strict
# frozen_string_literal: true

class Follow < ApplicationRecord
  belongs_to :source_profile, class_name: "Profile"
  belongs_to :target_profile, class_name: "Profile"

  sig { void }
  def check_suggested!
    SuggestedFollow.find_by(source_profile:, target_profile:)&.touch(:checked_at)
  end
end
