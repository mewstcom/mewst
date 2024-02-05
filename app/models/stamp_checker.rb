# typed: strict
# frozen_string_literal: true

class StampChecker
  extend T::Sig

  T::Sig::WithoutRuntime.sig { params(profile: T.nilable(Profile), posts: T.any(Post::PrivateRelation, T::Array[Post])).void }
  def initialize(profile:, posts:)
    @profile = profile
    @posts = posts
  end

  sig { params(post: Post).returns(T::Boolean) }
  def stamped?(post:)
    !profile.nil? && stamped_post_ids.include?(post.id)
  end

  sig { returns(T::Array[T::Mewst::DatabaseId]) }
  private def stamped_post_ids
    @stamped_post_ids ||= profile.stamps.where(post: posts).pluck(:post_id)
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  T::Sig::WithoutRuntime.sig { returns(T.any(Post::PrivateRelation, T::Array[Post])) }
  attr_reader :posts
  private :posts
end
