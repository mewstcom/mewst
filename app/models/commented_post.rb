# typed: strict
# frozen_string_literal: true

class CommentedPost < ApplicationRecord
  has_one :post, as: :postable, dependent: :restrict_with_exception, touch: true

  validates :comment, length: {maximum: Commentable::MAXIMUM_COMMENT_LENGTH}, presence: true
end
