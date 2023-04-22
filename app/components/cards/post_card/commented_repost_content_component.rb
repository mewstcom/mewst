# typed: strict
# frozen_string_literal: true

class Cards::PostCard::CommentedRepostContentComponent < ApplicationComponent
  sig { params(post: Post).void }
  def initialize(post:)
    @post = post
  end

  private

  sig { returns(Post) }
  attr_reader :post

  sig { returns(Profile) }
  def profile
    T.must(post.profile)
  end

  sig { returns(Repost) }
  def repost
    post.postable
  end

  sig { returns(Post) }
  def target_post
    repost.repostable.post
  end

  sig { returns(Profile) }
  def target_profile
    T.must(target_post.profile)
  end
end
