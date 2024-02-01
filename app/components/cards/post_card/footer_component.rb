# typed: strict
# frozen_string_literal: true

class Cards::PostCard::FooterComponent < ApplicationComponent
  sig { params(post: Post).void }
  def initialize(post:)
    @post = post
  end

  sig { returns(Post) }
  attr_reader :post
  private :post

  sig { returns(Profile) }
  private def profile
    post.profile.not_nil!
  end
end
