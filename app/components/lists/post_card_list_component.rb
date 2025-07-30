# typed: strict
# frozen_string_literal: true

class Lists::PostCardListComponent < ApplicationComponent
  sig do
    params(
      posts: T.any(ActiveRecord::Relation, T::Array[PostRecord]),
      stamp_checker: StampChecker,
      follow_checker: T.nilable(FollowChecker),
      class_name: String
    ).void
  end
  def initialize(posts:, stamp_checker:, follow_checker: nil, class_name: "")
    @posts = posts
    @stamp_checker = stamp_checker
    @follow_checker = follow_checker
    @class_name = class_name
  end

  sig { returns(T.any(ActiveRecord::Relation, T::Array[PostRecord])) }
  attr_reader :posts
  private :posts

  sig { returns(StampChecker) }
  attr_reader :stamp_checker
  private :stamp_checker

  sig { returns(T.nilable(FollowChecker)) }
  attr_reader :follow_checker
  private :follow_checker

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
