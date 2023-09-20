# typed: strict
# frozen_string_literal: true

class DeleteStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(profile: Profile, target_post: Post).returns(Result) }
  def call(profile:, target_post:)
    stamp = input.profile.stamps.where(post: input.target_post).sole
    stamp.destroy!

    Result.new(post: input.target_post.reload)
  end
end
