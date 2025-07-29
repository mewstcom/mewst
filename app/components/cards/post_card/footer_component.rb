# typed: strict
# frozen_string_literal: true

class Cards::PostCard::FooterComponent < ApplicationComponent
  sig { params(post: PostRecord, stamp_checker: StampChecker).void }
  def initialize(post:, stamp_checker:)
    @post = post
    @stamp_checker = stamp_checker
  end

  sig { returns(PostRecord) }
  attr_reader :post
  private :post

  sig { returns(StampChecker) }
  attr_reader :stamp_checker
  private :stamp_checker

  sig { returns(ProfileRecord) }
  private def profile
    post.profile_record.not_nil!
  end
end
