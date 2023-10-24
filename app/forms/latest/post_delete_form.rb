# typed: strict
# frozen_string_literal: true

class Latest::PostDeleteForm < Latest::ApplicationForm
  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  attribute :target_post_id, :string

  validates :viewer, presence: true
  validates :target_post, presence: true

  sig { returns(T.nilable(Post)) }
  def target_post
    viewer.posts.find_by(id: target_post_id)
  end
end
