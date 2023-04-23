# typed: strict
# frozen_string_literal: true

class Forms::Repost < Forms::Base
  attr_accessor :profile
  attribute :post_id, :string

  validates :post, presence: true

  private

  def post
    Post.find(post_id)
  end
end
