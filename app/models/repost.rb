# typed: strict
# frozen_string_literal: true

class Repost < ApplicationRecord
  counter_culture :comment_post

  belongs_to :comment_post
  belongs_to :follow
  belongs_to :post
  belongs_to :profile

  sig { returns(CommentPost) }
  def comment_post!
    T.must(comment_post)
  end

  sig { returns(Follow) }
  def follow!
    T.must(follow)
  end

  sig { returns(Post) }
  def post!
    T.must(post)
  end

  sig { returns(Profile) }
  def profile!
    T.must(profile)
  end

  sig { returns(Post) }
  def original_post
    comment_post!.post!
  end
end
