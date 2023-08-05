# typed: strict
# frozen_string_literal: true

class CommentPost < ApplicationRecord
  belongs_to :post

  validates :comment, length: {maximum: Commentable::MAXIMUM_COMMENT_LENGTH}, presence: true
end
