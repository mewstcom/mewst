# typed: strict
# frozen_string_literal: true

class Forms::CommentedPost < Forms::Base
  attr_accessor :profile
  attribute :comment, :string

  validates :comment, length: {maximum: Commentable::MAXIMUM_COMMENT_LENGTH}, presence: true
  validates :profile, presence: true
end
