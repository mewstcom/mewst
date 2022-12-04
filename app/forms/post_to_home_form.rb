# typed: strict
# frozen_string_literal: true

class PostToHomeForm < ApplicationForm
  attribute :post_id, :string
  attribute :profile_id, :string

  validates :post, presence: true
  validates :profile, presence: true

  sig { returns(T.nilable(Post)) }
  def post
    @post ||= T.let(Post.find_by(id: post_id), T.nilable(Post))
  end

  sig { returns(T.nilable(Profile)) }
  def profile
    @profile ||= T.let(Profile.find_by(id: profile_id), T.nilable(Profile))
  end
end
