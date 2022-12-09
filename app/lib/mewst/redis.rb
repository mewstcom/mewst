# typed: strict
# frozen_string_literal: true

class Mewst::Redis
  extend T::Sig

  sig { returns(T.any(ConnectionPool::Wrapper, Redis)) }
  def self.client
    @redis ||= T.let(ConnectionPool::Wrapper.new do
      Redis.new(url: ENV.fetch("MEWST_REDIS_CACHE_URL"))
    end, T.any(T.nilable(ConnectionPool::Wrapper), T.nilable(Redis)))
  end
end
