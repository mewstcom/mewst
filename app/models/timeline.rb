# typed: strict
# frozen_string_literal: true

class Timeline
  extend T::Sig

  class Order < T::Enum
    enums do
      Asc = new
      Desc = new
    end
  end

  sig { params(timeline_ownable: TimelineOwnable).void }
  def initialize(timeline_ownable)
    @timeline_ownable = timeline_ownable
  end

  sig { params(start_score: T.nilable(String), limit: Integer, order: Order).returns(T::Array[String]) }
  def post_ids(start_score: nil, limit: 50, order: Order::Desc)
    start, stop, rev = case order
    when Order::Asc
      [start_score.presence || "-inf", "+inf", false]
    when Order::Desc
      [start_score.presence || "+inf", "-inf", true]
    end

    redis_client.zrange(timeline_ownable.timeline_key, "(#{start}", "(#{stop}", by_score: true, rev:, limit: [0, limit])
  end

  private

  sig { returns(TimelineOwnable) }
  attr_reader :timeline_ownable

  sig { returns(Redis) }
  def redis_client
    T.cast(Mewst::Redis.new(url: Rails.configuration.mewst["redis_unevictable_cache_url"]).client, Redis)
  end
end
