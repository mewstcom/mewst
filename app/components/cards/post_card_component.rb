# typed: strict
# frozen_string_literal: true

class Cards::PostCardComponent < ApplicationComponent
  sig { params(post: Post).void }
  def initialize(post:)
    @post = post
    @profile = T.let(post.profile, Profile)
  end

  private

  sig { returns(Post) }
  attr_reader :post

  sig { returns(Profile) }
  attr_reader :profile
end
