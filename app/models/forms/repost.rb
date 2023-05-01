# typed: strict
# frozen_string_literal: true

class Forms::Repost < Forms::Base
  attr_accessor :profile
  attribute :post_id, :string

  validates :post, presence: true

  sig { returns(T.nilable(Post)) }
  def post
    Post.find_by(id: post_id)
  end
end
