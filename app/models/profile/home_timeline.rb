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

    posts = Post.where(id: T.must(post_ids).first(limit)).preload(:profile, :postable).order(id: :desc)

    has_page = ->(post_ids, limit) {
      post_ids.length == (limit + 1)
    }

    has_next_page, has_previous_page = if before_post.nil? && after_post.nil?
      [has_page.call(post_ids, limit), false]
    elsif before_post
      [true, has_page.call(post_ids, limit)]
    elsif after_post
      [has_page.call(post_ids, limit), true]
    end

    [posts, PageInfo.new(has_next_page:, has_previous_page:)]
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

  private

  sig { returns(Profile) }
  attr_reader :profile

  sig { returns(Redis) }
  def redis_client
    T.cast(Mewst::Redis.new(url: ENV.fetch("MEWST_REDIS_UNEVICTABLE_CACHE_URL")).client, Redis)
  end
end
