# typed: strict
# frozen_string_literal: true

class Profile::HomeTimeline
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  T::Sig::WithoutRuntime.sig do
    params(before: T.nilable(String), after: T.nilable(String), limit: Integer)
      .returns([Post::PrivateRelation, PageInfo])
  end
  def posts_with_page_info(before: nil, after: nil, limit: 50)
    before_post = before.nil? ? nil : Post.find(before)
    after_post = after.nil? ? nil : Post.find(after)

    post_ids = if before_post.nil? && after_post.nil?
      ::Timeline.new(profile).post_ids(
        start_score: nil,
        limit: limit + 1,
        order: ::Timeline::Order::Desc
      )
    elsif before_post
      ::Timeline.new(profile).post_ids(
        start_score: before_post.timeline_score,
        limit: limit + 1,
        order: ::Timeline::Order::Asc
      )
    elsif after_post
      ::Timeline.new(profile).post_ids(
        start_score: after_post.timeline_score,
        limit: limit + 1,
        order: ::Timeline::Order::Desc
      )
    end

    posts = Post.where(id: post_ids.not_nil!.first(limit)).preload(:profile).order(id: :desc)

    page_info = if before_post.nil? && after_post.nil?
      has_next_page = has_page(post_ids: post_ids.not_nil!, limit:)
      end_cursor = cursor(has_next_page:, post_ids: post_ids.not_nil!)
      has_previous_page = false
      start_cursor = nil

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif before_post
      has_next_page = true
      end_cursor = post_ids.first
      has_previous_page = has_page(post_ids:, limit:)
      start_cursor = cursor(has_next_page:, post_ids:)

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif after_post
      has_next_page = has_page(post_ids:, limit:)
      end_cursor = cursor(has_next_page:, post_ids:)
      has_previous_page = true
      start_cursor = post_ids.first

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    end

    [posts, page_info.not_nil!]
  end

  sig { params(post: Post).returns(T.self_type) }
  def add_post(post:)
    redis_client.zadd(profile.timeline_key, post.timeline_score, post.id)

    self
  end

  sig { params(post: Post).returns(T.self_type) }
  def remove_post(post:)
    redis_client.zrem(profile.timeline_key, post.id)

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
end
