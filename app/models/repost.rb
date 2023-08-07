# typed: strict
# frozen_string_literal: true

class Repost < ApplicationRecord
  counter_culture :original_comment_post, column_name: "reposts_count"

  belongs_to :original_follow, class_name: "Follow", foreign_key: "original_follow_id"
  belongs_to :original_post, class_name: "Post", foreign_key: "original_post_id"
  belongs_to :original_profile, class_name: "Profile", foreign_key: "original_profile_id"
  belongs_to :post
  belongs_to :target_post, class_name: "Post", foreign_key: "target_post_id", optional: true
  belongs_to :target_profile, class_name: "Profile", foreign_key: "target_profile_id", optional: true
  has_one :original_comment_post, source: :comment_post, through: :original_post

  sig { returns(Post) }
  def original_post!
    T.must(original_post)
  end

  sig { returns(Profile) }
  def original_profile!
    T.must(original_profile)
  end

  sig { returns(CommentPost) }
  def original_comment_post!
    original_post!.comment_post!
  end

  sig { returns(Post) }
  def target_post!
    T.must(target_post)
  end

  sig { returns(Profile) }
  def target_profile!
    T.must(target_profile)
  end
end
