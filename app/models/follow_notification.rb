# typed: strict
# frozen_string_literal: true

class FollowNotification < ApplicationRecord
  belongs_to :notification
  belongs_to :follow

  delegate :source_profile, :target_profile, to: :follow
end
