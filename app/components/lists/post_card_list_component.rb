# typed: strict
# frozen_string_literal: true

class Lists::PostCardListComponent < ApplicationComponent
  sig { params(posts: T.any(ActiveRecord::Relation, T::Array[Post]), stamp_checker: StampChecker).void }
  def initialize(posts:, stamp_checker:)
    @posts = posts
    @stamp_checker = stamp_checker
  end

  sig { returns(T.any(ActiveRecord::Relation, T::Array[Post])) }
  attr_reader :posts
  private :posts

  sig { returns(StampChecker) }
  attr_reader :stamp_checker
  private :stamp_checker
end
