# typed: strict
# frozen_string_literal: true

class StampChecker
  extend T::Sig

  sig { params(profile: T.nilable(Profile), posts: T.any(ActiveRecord::Relation, T::Array[Post])).void }
  def initialize(profile:, posts:)
    @profile = profile
    @posts = posts
    @stamped_post_ids = T.let(nil, T.nilable(T::Array[T::Mewst::DatabaseId]))
  end

  sig { params(post: Post).returns(T::Boolean) }
  def stamped?(post:)
    !profile.nil? && stamped_post_ids.include?(post.id)
  end

  sig { returns(T::Array[T::Mewst::DatabaseId]) }
  private def stamped_post_ids
    @stamped_post_ids ||= begin
      return [] if profile.nil? || posts.empty?

      profile.not_nil!.stamps.where(post: posts).pluck(:post_id)
    end
  end

  sig { returns(T.nilable(Profile)) }
  attr_reader :profile
  private :profile

  sig { returns(T.any(ActiveRecord::Relation, T::Array[Post])) }
  attr_reader :posts
  private :posts
end
