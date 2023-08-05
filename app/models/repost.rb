# typed: strict
# frozen_string_literal: true

class Repost < ApplicationRecord
  counter_culture :post, proc {|repost| repost.special? ? "reposts_count" : nil }

  belongs_to :original_follow, class_name: "Follow", foreign_key: "original_follow_id"
  belongs_to :original_post, class_name: "Post", foreign_key: "original_post_id"
  belongs_to :original_profile, class_name: "Profile", foreign_key: "original_profile_id"
  belongs_to :post
  belongs_to :target_post, class_name: "Post", foreign_key: "target_post_id", optional: true
  belongs_to :target_profile, class_name: "Profile", foreign_key: "target_profile_id", optional: true

  validates :comment, exclusion: { in: [nil] }, length: {maximum: Commentable::MAXIMUM_COMMENT_LENGTH}
end
