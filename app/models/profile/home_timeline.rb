# typed: strict
# frozen_string_literal: true

class ProfileRecord::HomeTimeline
  extend T::Sig

  sig { params(profile: ProfileRecord).void }
  def initialize(profile:)
    @profile = profile
  end

  T::Sig::WithoutRuntime.sig do
    params(after: T.nilable(String), before: T.nilable(String), limit: Integer).returns([PostRecord::PrivateRelation, PageInfo])
  end
  def fetch_posts(after: nil, before: nil, limit: 30)
    page = visible_posts.cursor_paginate(after:, before:, limit:, order: {published_at: :desc, id: :desc}).fetch
    page_info = PageInfo.from_cursor_paginate_page(page:)

    [page.records, page_info]
  end

  sig { params(post: PostRecord).returns(T.self_type) }
  def add_post!(post:)
    profile.home_timeline_posts.where(post:).first_or_create!(published_at: post.published_at)

    self
  end

  sig { returns(ProfileRecord) }
  attr_reader :profile
  private :profile

  T::Sig::WithoutRuntime.sig { returns(PostRecord::PrivateRelation) }
  private def visible_posts
    PostRecord.kept.preload(:profile, :link).joins(:home_timeline_posts).merge(profile.home_timeline_posts.visible)
  end
end
