# typed: strict
# frozen_string_literal: true

class Forms::CommentedPost < Forms::Base
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :comment, :string

  validates :comment, length: {maximum: Commentable::MAXIMUM_COMMENT_LENGTH}, presence: true
  validates :profile, presence: true

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end
end
