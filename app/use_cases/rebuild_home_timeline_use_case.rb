# typed: strict
# frozen_string_literal: true

class RebuildHomeTimelineUseCase < ApplicationUseCase
  class Result < T::Struct
    const :profile, T.nilable(Profile)
  end

  sig { params(profile_id: T::Mewst::DatabaseId, posts_limit: Integer).returns(Result) }
  def call(profile_id:, posts_limit:)
    profile = Profile.kept.find_by(id: profile_id)

    return Result.new(profile: nil) if profile.nil?

    profile.home_timeline.remove_all_posts

    profile.posts.kept.order(published_at: :desc).limit(posts_limit).find_each do |post|
      profile.home_timeline.add_post!(post:)
    end

    profile.followee_posts.kept.order(published_at: :desc).limit(posts_limit).find_each do |post|
      profile.home_timeline.add_post!(post:)
    end

    Result.new(profile:)
  end
end
