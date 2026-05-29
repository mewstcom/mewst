# typed: strict
# frozen_string_literal: true

class StampForm < ApplicationForm
  attribute :target_post_id, :string

  validates :target_post, presence: true

  sig { returns(T.nilable(PostRecord)) }
  def target_post
    PostRecord.find_by(id: target_post_id)
  end
end
