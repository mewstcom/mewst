# typed: strict
# frozen_string_literal: true

class PostForm < ApplicationForm
  attribute :content, :string
  attribute :with_frame, :boolean, default: false

  validates :content, length: {maximum: Post::MAXIMUM_CONTENT_LENGTH}, presence: true
end
