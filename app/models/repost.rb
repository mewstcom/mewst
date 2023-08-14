# typed: strict
# frozen_string_literal: true

class Repost < ApplicationRecord
  counter_culture :comment_post

  belongs_to :comment_post
  belongs_to :follow
  belongs_to :post
  belongs_to :profile

  sig { returns(Post) }
  def original_post
    comment_post.not_nil!.post.not_nil!
  end
end
