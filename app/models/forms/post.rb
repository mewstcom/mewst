# typed: strict
# frozen_string_literal: true

class Forms::Post < Forms::Base
  attr_accessor :profile
  attribute :post_id, :string

  validates :post, presence: true

  private

  def post
    Post.find(post_id)
  end
end
