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
    new(url: Rails.configuration.mewst["redis_unevictable_cache_url"]).client
  end

  sig { returns(T.any(ConnectionPool::Wrapper, Redis)) }
  def client
    @client ||= T.let(ConnectionPool::Wrapper.new do
      Redis.new(url: @url)
    end, T.any(T.nilable(ConnectionPool::Wrapper), T.nilable(Redis)))
  end
end
