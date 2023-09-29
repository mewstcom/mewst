# typed: strict
# frozen_string_literal: true

class Latest::PostForm < Latest::ApplicationForm
  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  attribute :comment, :string

  validates :comment, length: {maximum: Post::MAXIMUM_COMMENT_LENGTH}, presence: true
  validates :viewer, presence: true
end
