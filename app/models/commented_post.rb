# typed: strict
# frozen_string_literal: true

class CommentedPost < ApplicationRecord
  MAXIMUM_COMMENT_LENGTH = 500

  has_one :post, as: :postable, touch: true

  validates :comment, length: {maximum: MAXIMUM_COMMENT_LENGTH}, presence: true
end
