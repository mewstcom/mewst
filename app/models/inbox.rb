# typed: strict
# frozen_string_literal: true

class Inbox
  extend T::Sig

  sig { params(inboxable: Inboxable).void }
  def initialize(inboxable)
    @inboxable = inboxable
  end

  sig { params(limit: Integer, from_post: T.nilable(Post)).returns(T::Array[String]) }
  def get_post_ids(limit: 10, from_post: nil)
    start = "+inf"
    stop = "-inf"

    T.cast(Mewst::Redis.client, Redis).zrange(@inboxable.inbox_key, start, stop, by_score: true, rev: true, limit: [0, limit])
  end
end
