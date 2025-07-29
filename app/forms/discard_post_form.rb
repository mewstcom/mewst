# typed: strict
# frozen_string_literal: true

class DiscardPostForm < ApplicationForm
  sig { returns(T.nilable(ProfileRecord)) }
  attr_accessor :profile

  attribute :target_post_id, :string

  validates :profile, presence: true
  validates :target_post, presence: true

  sig { returns(T.nilable(PostRecord)) }
  def target_post
    profile&.posts&.kept&.find_by(id: target_post_id)
  end
end
