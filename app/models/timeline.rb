# typed: strict
# frozen_string_literal: true

class Timeline
  extend T::Sig

  sig { params(timeline_ownable: TimelineOwnable).void }
  def initialize(timeline_ownable)
    @timeline_ownable = timeline_ownable
  end

  sig { params(limit: Integer, from_post: T.nilable(Post)).returns(T::Array[String]) }
  def get_post_ids(limit: 10, from_post: nil)
    start = "+inf"
    stop = "-inf"

    T.cast(Mewst::Redis.client, Redis).zrange(timeline_ownable.timeline_key, start, stop, by_score: true, rev: true, limit: [0, limit])
  end

  private

  sig { returns(TimelineOwnable) }
  attr_reader :timeline_ownable
end
