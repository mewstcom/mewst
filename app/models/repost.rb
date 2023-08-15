# typed: strict
# frozen_string_literal: true

class Repost < ApplicationRecord
  counter_culture :target_comment_post

  belongs_to :follow
  belongs_to :post
  belongs_to :profile
  belongs_to :target_comment_post, class_name: "CommentPost"

  sig { returns(Post) }
  def original_post
    target_comment_post.not_nil!.post.not_nil!
  end
end
