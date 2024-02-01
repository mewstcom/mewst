# typed: strict
# frozen_string_literal: true

class V1::PostForm < V1::ApplicationForm
  sig { returns(T.nilable(Actor)) }
  attr_accessor :viewer

  attribute :content, :string

  validates :content, length: {maximum: Post::MAXIMUM_CONTENT_LENGTH}, presence: true
  validates :viewer, presence: true
end
