# typed: false
# frozen_string_literal: true

namespace :suggested_follow do
  desc "おすすめプロフィール (自分がフォローしている人がフォローしている人) を作成する"
  task create: :environment do
    # Note: limitに指定している数値に深い意味はない
    Profile.kept.sort_by_latest_post.limit(1_000).each do |source_profile|
      source_profile.create_suggested_follows!
    end
  end
end
