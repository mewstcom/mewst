# typed: strict
# frozen_string_literal: true

class Profile::Timeline::Home
  extend T::Sig

  class PageInfo < T::Struct
    const :end_cursor, T.nilable(String)
    const :has_next_page, T::Boolean
    const :has_previous_page, T::Boolean
    const :start_cursor, T.nilable(String)
  end

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
        start_score: before_post.timeline_item_score,
        limit: limit + 1,
        order: ::Timeline::Order::Asc
      )
    elsif after_post
      ::Timeline.new(profile).post_ids(
        start_score: after_post.timeline_item_score,
        limit: limit + 1,
        order: ::Timeline::Order::Desc
      )
    end

    posts = Post.where(id: T.must(post_ids).first(limit)).preload(:profile).order(id: :desc)

    has_page = ->(post_ids, limit) {
      post_ids.length == (limit + 1)
    }

    cursor = ->(has_page, post_ids) {
      has_page ? post_ids[-2] : post_ids.last
    }

    page_info = if before_post.nil? && after_post.nil?
      has_next_page = has_page.call(post_ids, limit)
      end_cursor = cursor.call(has_next_page, post_ids)
      has_previous_page = false
      start_cursor = nil

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif before_post
      has_next_page = true
      end_cursor = post_ids.first
      has_previous_page = has_page.call(post_ids, limit)
      start_cursor = cursor.call(has_next_page, post_ids)

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif after_post
      has_next_page = has_page.call(post_ids, limit)
      end_cursor = cursor.call(has_next_page, post_ids)
      has_previous_page = true
      start_cursor = post_ids.first
      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    end

    [posts, T.must(page_info)]
  end

  sig { params(post: Post).returns(T.self_type) }
  def add_post(post:)
    T.cast(Mewst::Redis.client, Redis).zadd(profile.timeline_key, post.timeline_item_score, post.id)

    self
  end

  sig { params(post: Post).returns(T.self_type) }
  def remove_post(post:)
    T.cast(Mewst::Redis.client, Redis).zrem(profile.timeline_key, post.id)

    self
  end

  private

  sig { returns(Profile) }
  attr_reader :profile
end
