# typed: strict
# frozen_string_literal: true

class Latest::PostForm < Latest::ApplicationForm
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :comment, :string

  validates :comment, length: {maximum: Post::MAXIMUM_COMMENT_LENGTH}, presence: true
  validates :profile, presence: true
end
