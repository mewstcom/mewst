# typed: strict
# frozen_string_literal: true

class StampChecker
  extend T::Sig

  T::Sig::WithoutRuntime.sig { params(profile: Profile, posts: Post::PrivateRelation).void }
  def initialize(profile:, posts:)
    @profile = profile
    @posts = posts
  end

  def stamped?(post:)
    stamped_post_ids.include?(post.id)
  end

  private def stamped_post_ids
    @stamped_post_ids ||= profile.stamps.where(post: posts).pluck(:post_id)
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  T::Sig::WithoutRuntime.sig { returns(Post::PrivateRelation) }
  attr_reader :posts
  private :posts
end
