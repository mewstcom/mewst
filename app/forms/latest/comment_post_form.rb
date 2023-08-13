# typed: strict
# frozen_string_literal: true

class Latest::CommentPostForm < Latest::ApplicationForm
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :comment, :string

  validates :comment, length: {maximum: Commentable::MAXIMUM_COMMENT_LENGTH}, presence: true
  validates :profile, presence: true

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end

  sig { returns(String) }
  def comment!
    T.must(comment)
  end
end
