# typed: strict
# frozen_string_literal: true

class V1::StampForm < V1::ApplicationForm
  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  attribute :target_post_id, :string

  validates :viewer, presence: true
  validates :target_post, presence: true

  sig { returns(T.nilable(Post)) }
  def target_post
    Post.find_by(id: target_post_id)
  end
end
