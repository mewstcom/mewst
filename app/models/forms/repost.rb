# typed: strict
# frozen_string_literal: true

class Forms::Repost < Forms::Base
  sig { returns(T.nilable(Profile)) }
  attr_accessor :profile

  attribute :post_id, :string

  validates :post, presence: true

  sig { returns(T.nilable(Post)) }
  def post
    Post.find_by(id: post_id)
  end

  sig { returns(Post) }
  def post!
    T.cast(post, Post)
  end

  sig { returns(Profile) }
  def profile!
    T.cast(profile, Profile)
  end
end
