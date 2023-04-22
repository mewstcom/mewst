# typed: strict
# frozen_string_literal: true

class CommentedRepost < ApplicationRecord
  delegated_type :repostable, types: Repostable::TYPES, dependent: :destroy

  counter_culture :repostable, column_name: "reposts_count"

  has_one :post, as: :postable, touch: true

  validates :comment, length: {maximum: Commentable::MAXIMUM_COMMENT_LENGTH}, presence: true
  validates :repostable_type, inclusion: {in: Repostable::TYPES}
end
