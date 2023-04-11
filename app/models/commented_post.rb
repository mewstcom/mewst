# typed: strict
# frozen_string_literal: true

class CommentedPost < ApplicationRecord
  MAXIMUM_COMMENT_LENGTH = 500

  validates :comment, length: {maximum: MAXIMUM_COMMENT_LENGTH}, presence: true
end
