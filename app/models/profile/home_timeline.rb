# typed: strict
# frozen_string_literal: true

class Profile::HomeTimeline
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  T::Sig::WithoutRuntime.sig do
    params(after: T.nilable(String), before: T.nilable(String)).returns(Post::PrivateRelation, PageInfo)
  end
  def fetch_posts(after: nil, before: nil, limit: 30)
    page = visible_posts.cursor_paginate(after:, before:, limit:, order: {published_at: :desc, id: :desc}).fetch
    page_info = PageInfo.from_cursor_paginate_page(page:)

    [page.records, page_info]
  end

  sig { params(post: Post).returns(T.self_type) }
  def add_post!(post:)
    profile.home_timeline_posts.where(post:).first_or_create!(published_at: post.published_at)

    self
  end

  sig { params(post: Post).returns(T.self_type) }
  def remove_post(post:)
    redis_client.zrem(profile.timeline_key, post.id)

    self
  end

  sig { returns(T.self_type) }
  def remove_all_posts
    redis_client.del(profile.timeline_key)

    self
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  sig { returns(Redis) }
  private def redis_client
    T.cast(Mewst::Redis.new(url: Rails.configuration.mewst["redis_url"]).client, Redis)
  end

  sig { params(post_ids: T::Array[String], limit: Integer).returns(T::Boolean) }
  private def has_page(post_ids:, limit:)
    post_ids.length == (limit + 1)
  end

  sig { params(has_next_page: T::Boolean, post_ids: T::Array[String]).returns(T.nilable(String)) }
  private def cursor(has_next_page:, post_ids:)
    has_next_page ? post_ids[-2] : post_ids.last
  end

  T::Sig::WithoutRuntime.sig { returns(Post::PrivateRelation) }
  private def visible_posts
    Post.kept.preload(:profile).joins(:home_timeline_posts).merge(profile.home_timeline_posts.visible)
  end
end
