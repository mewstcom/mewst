# typed: strict
# frozen_string_literal: true

class Cards::CommentedPostCardComponent < ApplicationComponent
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
end
