# typed: strict
# frozen_string_literal: true

class Latest::RepostForm < Latest::ApplicationForm
  sig { returns(T.nilable(Profile)) }
  attr_accessor :viewer

  attribute :target_post_id, :string

  validates :viewer, presence: true
  validates :target_post, presence: true
  validates :follow, presence: true

  sig { returns(Post) }
  def target_post!
    T.must(target_post)
  end

  sig { returns(Follow) }
  def follow!
    T.must(follow)
  end

  sig { returns(Profile) }
  def viewer!
    T.must(viewer)
  end

  sig { returns(T.nilable(Post)) }
  private def target_post
    Post.find_by(id: target_post_id)
  end

  sig { returns(Post) }
  private def original_post
    target_post!.original_post
  end

  sig { returns(T.nilable(Follow)) }
  private def follow
    return if target_post.nil?

    viewer!.follows.find_by(target_profile_id: original_post.profile_id)
  end
end
