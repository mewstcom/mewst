# typed: strict
# frozen_string_literal: true

class Mewst::Redis
  extend T::Sig

  sig { params(url: String).void }
  def initialize(url:)
    @url = url
  end

  sig { returns(T.any(ConnectionPool::Wrapper, Redis)) }
  def self.unevictable_client
    new(url: ENV.fetch("MEWST_REDIS_UNEVICTABLE_CACHE_URL")).client
  end

  sig { returns(T.any(ConnectionPool::Wrapper, Redis)) }
  def client
    @client ||= T.let(ConnectionPool::Wrapper.new do
      Redis.new(url: @url)
    end, T.any(T.nilable(ConnectionPool::Wrapper), T.nilable(Redis)))
  end
end
