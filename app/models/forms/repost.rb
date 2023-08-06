# typed: strict
# frozen_string_literal: true

class Forms::Repost < Forms::Base
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :target_post_id, :string

  validates :profile, presence: true
  validates :target_post, presence: true
  validates :original_follow, presence: true

  sig { returns(T.nilable(Follow)) }
  def original_follow
    return if target_post.nil?

    profile!.follows.find_by(target_profile_id: target_post!.original_post.profile_id)
  end

  sig { returns(Follow) }
  def original_follow!
    T.must(original_follow)
  end

  sig { returns(Post) }
  def original_post
    target_post!.original_post
  end

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(T.nilable(Post)) }
  def target_post
    Post.find_by(id: target_post_id)
  end

  sig { returns(Post) }
  def target_post!
    T.must(target_post)
  end
end
