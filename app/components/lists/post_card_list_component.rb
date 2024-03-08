# typed: strict
# frozen_string_literal: true

class Lists::PostCardListComponent < ApplicationComponent
  sig { params(posts: T.any(ActiveRecord::Relation, T::Array[Post]), stamp_checker: StampChecker, class_name: String).void }
  def initialize(posts:, stamp_checker:, class_name: "")
    @posts = posts
    @stamp_checker = stamp_checker
    @class_name = class_name
  end

  sig { returns(T.any(ActiveRecord::Relation, T::Array[Post])) }
  attr_reader :posts
  private :posts

  sig { returns(StampChecker) }
  attr_reader :stamp_checker
  private :stamp_checker

  sig { returns(String) }
  attr_reader :class_name
  private :class_name
end
