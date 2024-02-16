# typed: strict
# frozen_string_literal: true

class Cards::DetailedPostCardComponent < ApplicationComponent
  sig { params(post: Post, stamp_checker: StampChecker).void }
  def initialize(post:, stamp_checker:)
    @post = post
    @stamp_checker = stamp_checker
  end

  sig { returns(Post) }
  attr_reader :post
  private :post

  sig { returns(StampChecker) }
  attr_reader :stamp_checker
  private :stamp_checker

  sig { returns(Profile) }
  private def profile
    post.profile.not_nil!
  end
end
