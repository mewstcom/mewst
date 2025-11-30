# typed: strict
# frozen_string_literal: true

class Cards::PostCardComponent < ApplicationComponent
  sig { params(post: PostRecord, stamp_checker: StampChecker, follow_checker: T.nilable(FollowChecker)).void }
  def initialize(post:, stamp_checker:, follow_checker: nil)
    @post = post
    @stamp_checker = stamp_checker
    @follow_checker = follow_checker
  end

  sig { returns(PostRecord) }
  attr_reader :post
  private :post

  sig { returns(StampChecker) }
  attr_reader :stamp_checker
  private :stamp_checker

  sig { returns(T.nilable(FollowChecker)) }
  attr_reader :follow_checker
  private :follow_checker
end
