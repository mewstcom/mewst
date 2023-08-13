# typed: strict
# frozen_string_literal: true

class Latest::StampForm < Latest::ApplicationForm
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :target_post_id, :string

  validates :profile, presence: true
  validates :target_post, presence: true

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(Post) }
  def original_post
    target_post!.original_post
  end

  sig { returns(Post) }
  def target_post!
    T.must(target_post)
  end

  sig { returns(T.nilable(Post)) }
  private def target_post
    Post.find_by(id: target_post_id)
  end
end
