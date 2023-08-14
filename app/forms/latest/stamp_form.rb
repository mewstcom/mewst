# typed: strict
# frozen_string_literal: true

class Latest::StampForm < Latest::ApplicationForm
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :target_post_id, :string

  validates :profile, presence: true
  validates :target_post, presence: true

  sig { returns(Post) }
  def original_post
    target_post.not_nil!.original_post
  end

  sig { returns(T.nilable(Post)) }
  def target_post
    Post.find_by(id: target_post_id)
  end
end
