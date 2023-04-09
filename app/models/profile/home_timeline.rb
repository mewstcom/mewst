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
      .returns([Entry::PrivateRelation, PageInfo])
  end
  def entries_with_page_info(before: nil, after: nil, limit: 50)
    before_entry = before.nil? ? nil : Entry.find(before)
    after_entry = after.nil? ? nil : Entry.find(after)

    entry_ids = if before_entry.nil? && after_entry.nil?
      ::Timeline.new(profile).entry_ids(
        start_score: nil,
        limit: limit + 1,
        order: ::Timeline::Order::Desc
      )
    elsif before_entry
      ::Timeline.new(profile).entry_ids(
        start_score: before_entry.timeline_score,
        limit: limit + 1,
        order: ::Timeline::Order::Asc
      )
    elsif after_entry
      ::Timeline.new(profile).entry_ids(
        start_score: after_entry.timeline_score,
        limit: limit + 1,
        order: ::Timeline::Order::Desc
      )
    end

    entries = Entry.where(id: T.must(entry_ids).first(limit)).preload(:profile, :entryable).order(id: :desc)

    has_page = ->(entry_ids, limit) {
      entry_ids.length == (limit + 1)
    }

    cursor = ->(has_page, entry_ids) {
      has_page ? entry_ids[-2] : entry_ids.last
    }

    page_info = if before_entry.nil? && after_entry.nil?
      has_next_page = has_page.call(entry_ids, limit)
      end_cursor = cursor.call(has_next_page, entry_ids)
      has_previous_page = false
      start_cursor = nil

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif before_entry
      has_next_page = true
      end_cursor = entry_ids.first
      has_previous_page = has_page.call(entry_ids, limit)
      start_cursor = cursor.call(has_next_page, entry_ids)

      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    elsif after_entry
      has_next_page = has_page.call(entry_ids, limit)
      end_cursor = cursor.call(has_next_page, entry_ids)
      has_previous_page = true
      start_cursor = entry_ids.first
      PageInfo.new(end_cursor:, has_next_page:, has_previous_page:, start_cursor:)
    end

    [entries, T.must(page_info)]
  end

  sig { params(entry: Entry).returns(T.self_type) }
  def add_entry(entry:)
    redis_client.zadd(profile.timeline_key, entry.timeline_score, entry.id)

    self
  end

  sig { params(entry: Entry).returns(T.self_type) }
  def remove_entry(entry:)
    redis_client.zrem(profile.timeline_key, entry.id)

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
