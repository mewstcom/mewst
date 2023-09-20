# typed: strict
# frozen_string_literal: true

class CreateStampUseCase < ApplicationUseCase
  class Result < T::Struct
    const :post, Post
  end

  sig { params(profile: Profile, target_post: Post).returns(Result) }
  def call(profile:, target_post:)
    profile.stamps.where(post: target_post).first_or_create!(stamped_at: Time.current)

    Result.new(post: target_post.reload)
  end
end
