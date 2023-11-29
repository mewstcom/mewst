# typed: false
# frozen_string_literal: true

namespace :timeline do
  desc "タイムラインを生成し直す"
  task rebuild_for_active_profiles: :environment do
    profiles_limit = ENV.fetch("PROFILES_LIMIT", 1_000).to_i

    Profile.kept.sort_by_latest_post.limit(profiles_limit).find_each do |profile|
      posts_limit = ENV.fetch("POSTS_LIMIT", 1_000).to_i

      RebuildHomeTimelineJob.perform_later(profile_id: profile.id, posts_limit:)
    end
  end

  desc "特定プロフィールのタイムラインを生成し直す"
  task rebuild_for_specific_profile: :environment do
    profile_id = ENV.fetch("PROFILE_ID")
    posts_limit = ENV.fetch("POSTS_LIMIT", 1_000).to_i

    profile = Profile.kept.find(profile_id)

    RebuildHomeTimelineJob.perform_later(profile_id: profile.id, posts_limit:)
  end
end
