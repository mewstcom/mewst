# typed: strict
# frozen_string_literal: true

class Latest::PostForm < Latest::ApplicationForm
  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  attribute :content, :string

  validates :content, length: {maximum: Post::MAXIMUM_CONTENT_LENGTH}, presence: true
  validates :viewer, presence: true
end
